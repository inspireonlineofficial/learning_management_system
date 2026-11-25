package enrollments

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"lms-backend/internal/domain/courses"
	"lms-backend/internal/infrastructure/rustfs"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

const (
	streamURLTTL = 2 * time.Hour
	// maxPlaylistBytes bounds how much of a playlist object is read.
	maxPlaylistBytes = 2 << 20
)

var renditionNamePattern = regexp.MustCompile(`^v[0-9]{1,2}$`)

// objectReader is the optional storage capability needed to serve HLS
// playlists (the S3 client implements it).
type objectReader interface {
	GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, error)
}

// GetStreamingSignedURL generates playback URLs for a lesson video.
// Requirements: 13.3, 13.4, 13.5, 7.3
//
// Access rules:
//   - the lesson must resolve to a course and the attached video must belong
//     to that same course (a lesson cannot "borrow" another course's video);
//   - free / free-preview lessons of a published course are open to any
//     authenticated user;
//   - other lessons require a complete profile and an active enrollment;
//   - draft lessons and lessons in unpublished modules are never streamable
//     by students.
func (s *service) GetStreamingSignedURL(ctx context.Context, cmd GetStreamingSignedURLCommand) (*StreamingSignedURLResponse, error) {
	lesson, err := s.lessonRepo.FindByID(ctx, cmd.LessonID)
	if err != nil || lesson == nil {
		return nil, apperrors.NewNotFoundError("LESSON_NOT_FOUND", "lesson not found")
	}
	if lesson.VideoID == nil {
		return nil, apperrors.NewSimpleValidationError("NO_VIDEO", "this lesson does not have a video")
	}

	user, err := s.userRepo.FindByID(ctx, cmd.UserID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("USER_NOT_FOUND", "user not found")
	}

	video, err := s.videoRepo.FindByID(ctx, *lesson.VideoID)
	if err != nil || video == nil {
		return nil, apperrors.NewNotFoundError("VIDEO_NOT_FOUND", "video not found")
	}

	placement, err := s.resolveLessonCourse(ctx, lesson)
	if err != nil {
		return nil, err
	}
	if video.CourseID != placement.CourseID {
		return nil, apperrors.NewForbiddenError("ACCESS_DENIED", "you do not have access to this lesson")
	}
	if !isVisibleToStudents(lesson, placement) {
		return nil, apperrors.NewNotFoundError("LESSON_NOT_FOUND", "lesson is not available")
	}

	if lesson.IsFreePreview || lesson.IsFree {
		if placement.CourseStatus != courses.CourseStatusPublished {
			return nil, apperrors.NewNotFoundError("LESSON_NOT_FOUND", "lesson is not available")
		}
	} else {
		// Non-preview lessons require profile_complete check (Requirement 7.3)
		if !user.ProfileComplete {
			return nil, apperrors.ErrProfileIncomplete
		}
		// Check enrollment status at application service layer (Requirement 13.3/13.7)
		enrollment, err := s.enrollmentRepo.FindByStudentAndCourse(ctx, cmd.UserID, placement.CourseID)
		if err != nil || enrollment == nil {
			return nil, apperrors.NewForbiddenError("NOT_ENROLLED", "you must be enrolled in this course to access this lesson")
		}
		if !enrollment.CanAccess() {
			return nil, apperrors.NewForbiddenError("ENROLLMENT_REVOKED", "your enrollment has been cancelled or refunded")
		}
	}

	if video.Status != courses.VideoStatusReady {
		return nil, apperrors.NewSimpleValidationError("VIDEO_NOT_READY", "video is still processing or failed")
	}

	contentType := videoContentType(video.RustFSKey)
	// Cache for the full presign window so the browser can reuse byte-range
	// responses while the user scrubs/seeks. "private" keeps shared caches
	// (CDNs/proxies) from storing paid content under a reusable URL.
	signedURL, err := s.storageClient.PresignGetURLWithOptions(ctx, s.videoBucket, video.RustFSKey, streamURLTTL, rustfs.PresignOptions{
		ResponseContentType:        contentType,
		ResponseContentDisposition: "inline",
		ResponseCacheControl:       "private, max-age=7200, immutable",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	expiresAt := time.Now().Add(streamURLTTL)
	resp := &StreamingSignedURLResponse{
		SignedURL:   signedURL,
		ExpiresAt:   expiresAt,
		ContentType: contentType,
	}

	// HLS: the transcoder writes playlists with relative segment URIs, which
	// cannot be fetched from a private bucket through a single presigned
	// manifest URL. Instead we hand out a short-lived, HMAC-signed URL to the
	// backend playlist endpoint, which rewrites segment URIs to presigned URLs.
	if video.HLSManifestKey != "" && s.bunnyEnabled() {
		// Edge mode: the CDN validates the directory token and serves
		// cached playlists/segments; the bucket stays private.
		resp.HLSURL = bunnySignDirectory(s.edge.Host, s.edge.TokenKey, video.HLSManifestKey, time.Now().Add(s.edge.TTL).Unix())
		resp.HasHLS = true
		resp.Delivery = "edge"
	} else if video.HLSManifestKey != "" && len(s.streamSigningKey) > 0 {
		exp := expiresAt.Unix()
		resp.HLSURL = fmt.Sprintf("/v1/stream/hls/%s/master.m3u8?exp=%d&sig=%s", video.ID, exp, s.signHLS(video.ID, exp))
		resp.HasHLS = true
		resp.Delivery = "origin"
	}

	if video.ThumbnailRustFSKey != "" {
		posterURL, perr := s.storageClient.PresignGetURLWithOptions(ctx, s.videoBucket, video.ThumbnailRustFSKey, streamURLTTL, rustfs.PresignOptions{
			ResponseContentType:        "image/jpeg",
			ResponseContentDisposition: "inline",
			ResponseCacheControl:       "private, max-age=7200, immutable",
		})
		if perr == nil {
			resp.PosterURL = posterURL
		}
	}
	return resp, nil
}

func videoContentType(key string) string {
	switch strings.ToLower(path.Ext(key)) {
	case ".webm":
		return "video/webm"
	case ".mov":
		return "video/quicktime"
	default:
		return "video/mp4"
	}
}

func (s *service) signHLS(videoID uuid.UUID, exp int64) string {
	mac := hmac.New(sha256.New, s.streamSigningKey)
	fmt.Fprintf(mac, "hls|%s|%d", videoID, exp)
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *service) verifyHLS(videoID uuid.UUID, expRaw, sig string) (int64, error) {
	if len(s.streamSigningKey) == 0 {
		return 0, apperrors.NewNotFoundError("NOT_FOUND", "stream not available")
	}
	exp, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil || time.Now().Unix() > exp || exp > time.Now().Add(streamURLTTL+time.Minute).Unix() {
		return 0, apperrors.NewForbiddenError("STREAM_URL_EXPIRED", "stream URL has expired")
	}
	expected := s.signHLS(videoID, exp)
	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return 0, apperrors.NewForbiddenError("STREAM_URL_INVALID", "invalid stream signature")
	}
	return exp, nil
}

// GetHLSPlaylist serves a master (rendition == "") or rendition playlist for
// a signed HLS URL. Master playlists keep relative rendition URIs but carry the
// signature forward; rendition playlists get presigned segment URLs.
func (s *service) GetHLSPlaylist(ctx context.Context, videoID uuid.UUID, rendition, expRaw, sig string) ([]byte, error) {
	exp, err := s.verifyHLS(videoID, expRaw, sig)
	if err != nil {
		return nil, err
	}
	if rendition != "" && !renditionNamePattern.MatchString(rendition) {
		return nil, apperrors.NewNotFoundError("NOT_FOUND", "playlist not found")
	}
	reader, ok := s.storageClient.(objectReader)
	if !ok {
		return nil, apperrors.NewNotFoundError("NOT_FOUND", "stream not available")
	}
	video, err := s.videoRepo.FindByID(ctx, videoID)
	if err != nil || video == nil || video.HLSManifestKey == "" || video.Status != courses.VideoStatusReady {
		return nil, apperrors.NewNotFoundError("NOT_FOUND", "playlist not found")
	}

	baseDir := path.Dir(video.HLSManifestKey)
	key := video.HLSManifestKey
	if rendition != "" {
		key = path.Join(baseDir, rendition, "playlist.m3u8")
	}
	body, err := reader.GetObject(ctx, s.videoBucket, key)
	if err != nil {
		return nil, apperrors.NewNotFoundError("NOT_FOUND", "playlist not found")
	}
	defer body.Close()
	raw, err := io.ReadAll(io.LimitReader(body, maxPlaylistBytes))
	if err != nil {
		return nil, fmt.Errorf("read playlist: %w", err)
	}

	query := url.Values{}
	query.Set("exp", strconv.FormatInt(exp, 10))
	query.Set("sig", sig)
	ttl := time.Until(time.Unix(exp, 0)) + 5*time.Minute

	var out bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), maxPlaylistBytes)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			out.WriteString(line)
			out.WriteByte('\n')
			continue
		}
		if rendition == "" {
			// Variant playlist reference, e.g. "v0/playlist.m3u8".
			name := strings.SplitN(strings.TrimPrefix(line, "./"), "/", 2)[0]
			if !renditionNamePattern.MatchString(name) {
				continue
			}
			out.WriteString(name + "/playlist.m3u8?" + query.Encode())
			out.WriteByte('\n')
			continue
		}
		// Segment URI relative to the rendition directory.
		if strings.Contains(line, "..") || strings.Contains(line, "://") {
			continue // only plain relative segment names are expected
		}
		segment := path.Clean("/" + line)[1:]
		segKey := path.Join(baseDir, rendition, segment)
		signed, err := s.storageClient.PresignGetURLWithOptions(ctx, s.videoBucket, segKey, ttl, rustfs.PresignOptions{
			ResponseContentType:  "video/mp2t",
			ResponseCacheControl: "private, max-age=7200, immutable",
		})
		if err != nil {
			return nil, fmt.Errorf("presign segment: %w", err)
		}
		out.WriteString(signed)
		out.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parse playlist: %w", err)
	}
	return out.Bytes(), nil
}
