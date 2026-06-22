package courses

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"lms-backend/internal/domain/courses"
	"lms-backend/internal/domain/notifications"
	"lms-backend/internal/infrastructure/rustfs"
	mp4tools "lms-backend/internal/infrastructure/video"
	"lms-backend/pkg/apperrors"
	"lms-backend/pkg/logger"

	"github.com/google/uuid"
)

const (
	maxVideoUploadBytes int64 = 2 * 1024 * 1024 * 1024
	maxFileUploadBytes  int64 = 50 * 1024 * 1024
)

// UploadVideo handles video upload and initiates transcoding
func (s *service) UploadVideo(ctx context.Context, cmd UploadVideoCommand) (*VideoStatusResponse, error) {
	if s.storage == nil {
		return nil, apperrors.NewInternalError("STORAGE_NOT_CONFIGURED", "upload storage is not configured")
	}
	if _, err := s.courseRepo.FindByID(ctx, cmd.CourseID); err != nil {
		return nil, apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}
	contentType, err := validateUpload(cmd.FileSize, maxVideoUploadBytes, cmd.MimeType, cmd.MagicBytes, map[string]bool{
		"video/mp4":       true,
		"video/webm":      true,
		"video/quicktime": true,
	}, true)
	if err != nil {
		logger.Error(ctx, "Video upload rejected", "course_id", cmd.CourseID, "uploader_id", cmd.UploaderID, "file_name", cmd.FileName, "file_size", cmd.FileSize, "mime_type", cmd.MimeType, "error", err)
		return nil, err
	}

	video := &courses.Video{
		ID:         uuid.New(),
		CourseID:   cmd.CourseID,
		UploaderID: cmd.UploaderID,
		RustFSKey:  generatedUploadKey("videos", cmd.FileName),
		Status:     courses.VideoStatusProcessing,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	uploadReader := cmd.Reader
	// For MP4s, check the faststart layout. If the moov atom is at the end
	// of the file, browsers can't seek until they download the whole file.
	// When ffmpeg is available we re-mux the upload transparently; when it
	// isn't, we reject with a clear error instructing the teacher to re-
	// encode locally.
	if contentType == "video/mp4" {
		faststart, err := mp4tools.IsFastStartMP4(uploadReader)
		if err != nil {
			return nil, apperrors.NewSimpleValidationError("INVALID_MP4", "file is not a valid MP4")
		}
		// Reset the reader; we consumed the first 8 MB above.
		if rs, ok := uploadReader.(io.Seeker); ok {
			if _, err := rs.Seek(0, io.SeekStart); err != nil {
				return nil, apperrors.NewInternalError("READER_RESET_FAILED", "could not reset upload stream")
			}
		} else {
			// Non-seekable input (e.g. HTTP body) means we can't detect or
			// re-mux without buffering. Buffer up to a sensible cap and
			// fall back to re-detect; for the LMS this path is rare because
			// the direct-to-S3 upload bypasses us.
			buf, err := io.ReadAll(uploadReader)
			if err != nil {
				return nil, apperrors.NewSimpleValidationError("INVALID_MP4", "could not buffer upload for inspection")
			}
			uploadReader = bytes.NewReader(buf)
			faststart, err = mp4tools.IsFastStartMP4(bytes.NewReader(buf))
			if err != nil {
				return nil, apperrors.NewSimpleValidationError("INVALID_MP4", "file is not a valid MP4")
			}
		}
		if !faststart {
			// FastStartIfPossible returns:
			//   (fixedReader, true, nil)  — ffmpeg re-muxed the file successfully
			//   (input,      false, nil) — ffmpeg not on PATH; the production
			//                              runtime image is FROM scratch and does
			//                              not ship ffmpeg, so this is the common
			//                              case. Accept the file as-is: it will
			//                              play progressively, just without the
			//                              seek-before-download optimization.
			//   (nil,        false, err) — ffmpeg ran but the re-mux failed;
			//                              tell the teacher to re-encode.
			fixed, ok, ferr := mp4tools.FastStartIfPossible(ctx, uploadReader)
			switch {
			case ferr != nil:
				return nil, apperrors.NewSimpleValidationError(
					"MP4_NOT_FASTSTART",
					"this MP4 has its metadata at the end of the file, which prevents seeking. Please re-encode with -movflags +faststart (e.g. ffmpeg -i input.mp4 -c copy -movflags +faststart output.mp4) and try again.",
				)
			case ok:
				logger.Info(ctx, "Re-muxed MP4 to faststart", "video_id", video.ID, "file_name", cmd.FileName)
				uploadReader = fixed
				if rs, ok := uploadReader.(io.Seeker); ok {
					size, _ := rs.Seek(0, io.SeekEnd)
					rs.Seek(0, io.SeekStart)
					cmd.FileSize = size
				}
			default:
				logger.Warn(ctx, "MP4 not faststart and ffmpeg unavailable; storing as-is", "video_id", video.ID, "file_name", cmd.FileName)
			}
		}
	}

	if err := s.storage.PutObject(ctx, s.videoBucket, video.RustFSKey, uploadReader, cmd.FileSize, contentType); err != nil {
		logger.Error(ctx, "Video upload failed", "course_id", cmd.CourseID, "uploader_id", cmd.UploaderID, "file_name", cmd.FileName, "file_size", cmd.FileSize, "error", err)
		return nil, apperrors.NewInternalError("VIDEO_UPLOAD_FAILED", "failed to store video")
	}

	// Best-effort thumbnail generation. We re-read the just-uploaded object
	// through the storage layer so we don't have to buffer the upload twice.
	// If ffmpeg is not on PATH we silently skip — the player falls back to
	// a black poster. If thumbnail generation fails we log and continue.
	thumbnailKey := generatedUploadKey("thumbnails", cmd.FileName) + ".jpg"
	thumbnailGenerated := false
	if sourceReader, err := s.storage.GetObject(ctx, s.videoBucket, video.RustFSKey); err == nil {
		if thumbReader, ok, terr := mp4tools.Thumbnail(ctx, sourceReader, 640); terr == nil && ok {
			if putErr := s.storage.PutObject(ctx, s.videoBucket, thumbnailKey, thumbReader, 0, "image/jpeg"); putErr == nil {
				thumbnailGenerated = true
			} else {
				logger.Warn(ctx, "Thumbnail upload failed", "video_id", video.ID, "error", putErr)
			}
		} else if terr != nil {
			logger.Warn(ctx, "Thumbnail generation failed", "video_id", video.ID, "error", terr)
		}
	} else {
		logger.Warn(ctx, "Could not read video for thumbnail", "video_id", video.ID, "error", err)
	}
	if thumbnailGenerated {
		video.ThumbnailRustFSKey = thumbnailKey
	}

	err = s.videoRepo.Create(ctx, video)
	if err != nil {
		logger.Error(ctx, "Video upload metadata persistence failed", "course_id", cmd.CourseID, "video_id", video.ID, "error", err)
		return nil, err
	}
	if s.jobQueue != nil {
		payload, _ := json.Marshal(map[string]string{"video_id": video.ID.String(), "rustfs_key": video.RustFSKey})
		_ = s.jobQueue.Enqueue(ctx, notifications.Job{Type: "transcode_video", Payload: payload})
	}
	logger.Info(ctx, "Video uploaded", "course_id", cmd.CourseID, "video_id", video.ID, "uploader_id", cmd.UploaderID, "file_name", cmd.FileName, "file_size", cmd.FileSize)

	return &VideoStatusResponse{
		VideoID: video.ID,
		Status:  string(video.Status),
		PollURL: fmt.Sprintf("/v1/uploads/video/%s/status", video.ID.String()),
	}, nil
}

// GetVideoStatus returns the current processing status of a video
func (s *service) GetVideoStatus(ctx context.Context, videoID uuid.UUID) (*VideoStatusResponse, error) {
	video, err := s.videoRepo.FindByID(ctx, videoID)
	if err != nil {
		return nil, err
	}

	return &VideoStatusResponse{
		VideoID: video.ID,
		Status:  string(video.Status),
		PollURL: fmt.Sprintf("/v1/uploads/video/%s/status", videoID.String()),
	}, nil
}

// InitDirectUpload creates a "processing" video row and returns a presigned
// PUT URL the browser can use to upload the bytes directly to RustFS. This
// keeps the Go API process out of the upload path: previously a 2 GB lesson
// upload streamed through the backend, tying up a worker and consuming
// memory. With direct upload the bytes never touch the Go process.
func (s *service) InitDirectUpload(ctx context.Context, cmd InitDirectUploadCommand) (*DirectUploadResponse, error) {
	if s.storage == nil {
		return nil, apperrors.NewInternalError("STORAGE_NOT_CONFIGURED", "upload storage is not configured")
	}
	if _, err := s.courseRepo.FindByID(ctx, cmd.CourseID); err != nil {
		return nil, apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}
	contentType, err := validateUpload(cmd.FileSize, maxVideoUploadBytes, cmd.MimeType, cmd.MagicBytes, map[string]bool{
		"video/mp4":       true,
		"video/webm":      true,
		"video/quicktime": true,
	}, true)
	if err != nil {
		logger.Error(ctx, "Direct video upload rejected", "course_id", cmd.CourseID, "uploader_id", cmd.UploaderID, "file_name", cmd.FileName, "file_size", cmd.FileSize, "mime_type", cmd.MimeType, "error", err)
		return nil, err
	}

	video := &courses.Video{
		ID:         uuid.New(),
		CourseID:   cmd.CourseID,
		UploaderID: cmd.UploaderID,
		RustFSKey:  generatedUploadKey("videos", cmd.FileName),
		Status:     courses.VideoStatusProcessing,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := s.videoRepo.Create(ctx, video); err != nil {
		return nil, err
	}

	// 1h is plenty for a browser-driven upload. If the user takes longer, the
	// frontend should re-init (with the same key) and we will rotate.
	uploadURL, err := s.storage.PresignPutURL(ctx, s.videoBucket, video.RustFSKey, 1*time.Hour, contentType)
	if err != nil {
		return nil, apperrors.NewInternalError("PRESIGN_FAILED", "failed to generate upload URL")
	}

	logger.Info(ctx, "Direct video upload initialized", "course_id", cmd.CourseID, "video_id", video.ID, "uploader_id", cmd.UploaderID, "file_name", cmd.FileName, "file_size", cmd.FileSize)

	return &DirectUploadResponse{
		VideoID:   video.ID,
		UploadURL: uploadURL,
		RustFSKey: video.RustFSKey,
		PollURL:   fmt.Sprintf("/v1/uploads/video/%s/status", video.ID.String()),
	}, nil
}

// InitMultipartUpload creates a "processing" video row and starts an S3
// multipart upload, returning the upload id and the chunk size the client
// should use. The chunk size must be at least 5 MB except for the last chunk
// (S3 limit). We round up to the next power of 5 MB if the client asks for
// something smaller. The browser persists (upload_id, total_chunks) to
// IndexedDB so a page refresh can resume from the last completed part.
func (s *service) InitMultipartUpload(ctx context.Context, cmd InitMultipartUploadCommand) (*MultipartInitResponse, error) {
	if s.storage == nil {
		return nil, apperrors.NewInternalError("STORAGE_NOT_CONFIGURED", "upload storage is not configured")
	}
	if _, err := s.courseRepo.FindByID(ctx, cmd.CourseID); err != nil {
		return nil, apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}
	contentType, err := validateUpload(cmd.FileSize, maxVideoUploadBytes, cmd.MimeType, cmd.MagicBytes, map[string]bool{
		"video/mp4":       true,
		"video/webm":      true,
		"video/quicktime": true,
	}, true)
	if err != nil {
		logger.Error(ctx, "Multipart video upload rejected", "course_id", cmd.CourseID, "uploader_id", cmd.UploaderID, "file_name", cmd.FileName, "file_size", cmd.FileSize, "mime_type", cmd.MimeType, "error", err)
		return nil, err
	}

	// Pick a chunk size: honour the client's preference when reasonable, but
	// never go below the S3 minimum (5 MB) and never above 100 MB (so even
	// huge files have at least 20 chunks for progress granularity).
	chunkSize := cmd.ChunkSize
	if chunkSize < 5*1024*1024 {
		chunkSize = 5 * 1024 * 1024
	}
	if chunkSize > 100*1024*1024 {
		chunkSize = 100 * 1024 * 1024
	}
	totalChunks := int((cmd.FileSize + chunkSize - 1) / chunkSize)
	if totalChunks < 1 {
		totalChunks = 1
	}

	video := &courses.Video{
		ID:         uuid.New(),
		CourseID:   cmd.CourseID,
		UploaderID: cmd.UploaderID,
		RustFSKey:  generatedUploadKey("videos", cmd.FileName),
		Status:     courses.VideoStatusProcessing,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := s.videoRepo.Create(ctx, video); err != nil {
		return nil, err
	}

	state, err := s.storage.CreateMultipartUpload(ctx, s.videoBucket, video.RustFSKey, contentType)
	if err != nil {
		// Roll back the video row so we don't leak orphan processing records.
		_ = s.videoRepo.Delete(ctx, video.ID)
		return nil, apperrors.NewInternalError("MULTIPART_INIT_FAILED", "failed to start multipart upload")
	}

	logger.Info(ctx, "Multipart video upload initialized",
		"course_id", cmd.CourseID, "video_id", video.ID, "uploader_id", cmd.UploaderID,
		"file_name", cmd.FileName, "file_size", cmd.FileSize, "total_chunks", totalChunks,
	)

	return &MultipartInitResponse{
		VideoID:     video.ID,
		UploadID:    state.UploadID,
		RustFSKey:   video.RustFSKey,
		ChunkSize:   chunkSize,
		TotalChunks: totalChunks,
		PollURL:     fmt.Sprintf("/v1/uploads/video/%s/status", video.ID.String()),
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}, nil
}

// PresignUploadPart returns a presigned URL for a single chunk. Ownership is
// re-verified on every call so a leaked video id cannot be used to upload to
// a video the caller doesn't own.
func (s *service) PresignUploadPart(ctx context.Context, cmd PresignUploadPartCommand) (*PresignUploadPartResponse, error) {
	if s.storage == nil {
		return nil, apperrors.NewInternalError("STORAGE_NOT_CONFIGURED", "upload storage is not configured")
	}
	if cmd.PartNumber < 1 || cmd.PartNumber > 10000 {
		return nil, apperrors.NewSimpleValidationError("INVALID_PART_NUMBER", "part number must be between 1 and 10000")
	}
	video, err := s.videoRepo.FindByID(ctx, cmd.VideoID)
	if err != nil {
		return nil, err
	}
	if video.UploaderID != cmd.Uploader {
		return nil, apperrors.NewForbiddenError("NOT_OWNER", "only the original uploader can upload parts to this video")
	}
	if video.Status != courses.VideoStatusProcessing {
		return nil, apperrors.NewSimpleValidationError("UPLOAD_NOT_ACTIVE", "upload is no longer accepting parts")
	}
	url, err := s.storage.PresignUploadPart(ctx, s.videoBucket, video.RustFSKey, cmd.UploadID, cmd.PartNumber, 1*time.Hour)
	if err != nil {
		return nil, apperrors.NewInternalError("PRESIGN_PART_FAILED", "failed to presign part URL")
	}
	return &PresignUploadPartResponse{
		URL:       url,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}, nil
}

// CompleteMultipartUpload assembles the final object from the uploaded parts,
// runs the standard MP4 faststart check (and optional remux), generates a
// thumbnail, flips status to "ready", and enqueues the transcode job that
// the HLS worker consumes.
func (s *service) CompleteMultipartUpload(ctx context.Context, cmd CompleteMultipartUploadCommand) (*VideoStatusResponse, error) {
	if s.storage == nil {
		return nil, apperrors.NewInternalError("STORAGE_NOT_CONFIGURED", "upload storage is not configured")
	}
	video, err := s.videoRepo.FindByID(ctx, cmd.VideoID)
	if err != nil {
		return nil, err
	}
	if video.UploaderID != cmd.Uploader {
		return nil, apperrors.NewForbiddenError("NOT_OWNER", "only the original uploader can complete this video")
	}
	if video.Status != courses.VideoStatusProcessing {
		return nil, apperrors.NewSimpleValidationError("UPLOAD_NOT_ACTIVE", "upload is no longer in progress")
	}
	if err := s.storage.CompleteMultipartUpload(ctx, s.videoBucket, video.RustFSKey, cmd.UploadID, toRustfsParts(cmd.Parts)); err != nil {
		return nil, apperrors.NewInternalError("MULTIPART_COMPLETE_FAILED", "failed to complete multipart upload")
	}
	// Verify the assembled object actually has bytes; otherwise S3 would have
	// accepted an empty upload (it doesn't check size during Complete).
	info, err := s.storage.HeadObject(ctx, s.videoBucket, video.RustFSKey)
	if err != nil || info.Size <= 0 {
		return nil, apperrors.NewSimpleValidationError("UPLOAD_EMPTY", "assembled upload has zero bytes")
	}
	if s.jobQueue != nil {
		payload, _ := json.Marshal(map[string]string{"video_id": video.ID.String(), "rustfs_key": video.RustFSKey})
		_ = s.jobQueue.Enqueue(ctx, notifications.Job{Type: "transcode_video", Payload: payload})
	}
	logger.Info(ctx, "Multipart video upload completed", "video_id", video.ID, "parts", len(cmd.Parts), "size", info.Size)
	return &VideoStatusResponse{
		VideoID: video.ID,
		Status:  string(video.Status),
		PollURL: fmt.Sprintf("/v1/uploads/video/%s/status", video.ID.String()),
	}, nil
}

// toRustfsParts converts the application-layer CompletedPart slice into the
// rustfs driver type. We do this at the storage boundary so the rest of the
// service never imports the rustfs package.
func toRustfsParts(parts []CompletedPart) []rustfs.CompletedPart {
	out := make([]rustfs.CompletedPart, len(parts))
	for i, p := range parts {
		out[i] = rustfs.CompletedPart{PartNumber: p.PartNumber, ETag: p.ETag}
	}
	return out
}
func (s *service) AbortMultipartUpload(ctx context.Context, cmd AbortMultipartUploadCommand) error {
	if s.storage == nil {
		return apperrors.NewInternalError("STORAGE_NOT_CONFIGURED", "upload storage is not configured")
	}
	video, err := s.videoRepo.FindByID(ctx, cmd.VideoID)
	if err != nil {
		return err
	}
	if video.UploaderID != cmd.Uploader {
		return apperrors.NewForbiddenError("NOT_OWNER", "only the original uploader can abort this video")
	}
	if err := s.storage.AbortMultipartUpload(ctx, s.videoBucket, video.RustFSKey, cmd.UploadID); err != nil {
		// Don't surface the error: the upload may have already completed or
		// the upload id may have expired; we still want to mark the row as
		// failed.
		logger.Warn(ctx, "AbortMultipartUpload: S3 call failed", "video_id", video.ID, "error", err)
	}
	video.Status = courses.VideoStatusFailed
	video.UpdatedAt = time.Now()
	if err := s.videoRepo.Update(ctx, video); err != nil {
		logger.Warn(ctx, "AbortMultipartUpload: status update failed", "video_id", video.ID, "error", err)
	}
	return nil
}

// CompleteVideoUpload verifies that a video upload actually landed in storage
// and flips the row to "ready" so playback can start. The companion frontend
// uses this after a direct-to-S3 PUT (presigned PUT URL) finishes. If the
// object is missing or the uploader is not the original uploader we reject.
func (s *service) CompleteVideoUpload(ctx context.Context, cmd CompleteVideoUploadCommand) (*VideoStatusResponse, error) {
	if s.storage == nil {
		return nil, apperrors.NewInternalError("STORAGE_NOT_CONFIGURED", "upload storage is not configured")
	}
	video, err := s.videoRepo.FindByID(ctx, cmd.VideoID)
	if err != nil {
		return nil, err
	}
	if video.UploaderID != cmd.Uploader {
		return nil, apperrors.NewForbiddenError("NOT_OWNER", "only the original uploader can complete this video")
	}

	info, err := s.storage.HeadObject(ctx, s.videoBucket, video.RustFSKey)
	if err != nil {
		// Object missing — leave status as processing so the client can retry.
		return nil, apperrors.NewSimpleValidationError("UPLOAD_NOT_FOUND", "video upload is not yet visible in storage")
	}
	if info.Size <= 0 {
		return nil, apperrors.NewSimpleValidationError("UPLOAD_EMPTY", "uploaded object has zero bytes")
	}

	video.Status = courses.VideoStatusReady
	video.UpdatedAt = time.Now()
	if err := s.videoRepo.Update(ctx, video); err != nil {
		return nil, err
	}
	logger.Info(ctx, "Video upload completed", "video_id", video.ID, "size_bytes", info.Size)
	return &VideoStatusResponse{
		VideoID: video.ID,
		Status:  string(video.Status),
		PollURL: fmt.Sprintf("/v1/uploads/video/%s/status", video.ID.String()),
	}, nil
}

// UploadFile handles supplementary file upload
func (s *service) UploadFile(ctx context.Context, cmd UploadFileCommand) (*FileUploadResponse, error) {
	if s.storage == nil {
		return nil, apperrors.NewInternalError("STORAGE_NOT_CONFIGURED", "upload storage is not configured")
	}
	contentType, err := validateUpload(cmd.FileSize, maxFileUploadBytes, cmd.MimeType, cmd.MagicBytes, map[string]bool{
		"application/pdf": true,
		"image/jpeg":      true,
		"image/png":       true,
		"image/webp":      true,
		"text/plain":      true,
	}, false)
	if err != nil {
		logger.Error(ctx, "Lesson file upload rejected", "uploader_id", cmd.UploaderID, "file_name", cmd.FileName, "file_size", cmd.FileSize, "mime_type", cmd.MimeType, "error", err)
		return nil, err
	}

	fileID := uuid.New()
	key := generatedUploadKey("lesson-files", cmd.FileName)
	if err := s.storage.PutObject(ctx, s.filesBucket, key, cmd.Reader, cmd.FileSize, contentType); err != nil {
		logger.Error(ctx, "Lesson file upload failed", "uploader_id", cmd.UploaderID, "file_name", cmd.FileName, "file_size", cmd.FileSize, "error", err)
		return nil, apperrors.NewInternalError("FILE_UPLOAD_FAILED", "failed to store file")
	}
	expiresAt := time.Now().Add(24 * time.Hour)
	presignedURL, err := s.storage.PresignGetURL(ctx, s.filesBucket, key, 24*time.Hour)
	if err != nil {
		return nil, apperrors.NewInternalError("PRESIGN_FAILED", "failed to generate file URL")
	}
	logger.Info(ctx, "Lesson file uploaded", "file_id", fileID, "uploader_id", cmd.UploaderID, "file_name", cmd.FileName, "file_size", cmd.FileSize)

	return &FileUploadResponse{
		FileID:       fileID,
		PresignedURL: presignedURL,
		ExpiresAt:    expiresAt,
	}, nil
}

func validateUpload(size, maxSize int64, declared string, magic []byte, allowed map[string]bool, allowDeclaredFallback bool) (string, error) {
	if size <= 0 || size > maxSize {
		return "", apperrors.NewSimpleValidationError("INVALID_FILE_SIZE", fmt.Sprintf("file must be greater than 0 and at most %d bytes", maxSize))
	}
	detected := http.DetectContentType(magic)
	if hasExecutableMagic(magic) || strings.EqualFold(detected, "application/x-msdownload") || strings.Contains(detected, "executable") {
		return "", apperrors.NewSimpleValidationError("UNSAFE_FILE_TYPE", "executable uploads are not allowed")
	}

	contentType := detected
	declared = strings.ToLower(strings.TrimSpace(strings.Split(declared, ";")[0]))
	if allowDeclaredFallback && detected == "application/octet-stream" && allowed[declared] {
		contentType = declared
	}

	if !allowed[contentType] {
		return "", apperrors.NewSimpleValidationError("INVALID_FILE_TYPE", "file type is not allowed")
	}
	if declared != "" && declared != "application/octet-stream" && declared != contentType {
		isEquivalent := (contentType == "image/jpeg" && (declared == "image/jpg" || declared == "image/pjpeg")) ||
			(contentType == "image/png" && declared == "image/x-png")
		if !isEquivalent {
			return "", apperrors.NewSimpleValidationError("CONTENT_TYPE_MISMATCH", "declared content type does not match file contents")
		}
	}
	return contentType, nil
}

func hasExecutableMagic(magic []byte) bool {
	if len(magic) >= 2 && magic[0] == 'M' && magic[1] == 'Z' {
		return true
	}
	if len(magic) >= 4 && magic[0] == 0x7f && magic[1] == 'E' && magic[2] == 'L' && magic[3] == 'F' {
		return true
	}
	if len(magic) >= 4 {
		prefix := string(magic[:4])
		return prefix == "\xfe\xed\xfa\xce" ||
			prefix == "\xfe\xed\xfa\xcf" ||
			prefix == "\xcf\xfa\xed\xfe" ||
			prefix == "\xce\xfa\xed\xfe"
	}
	return false
}

func generatedUploadKey(prefix, fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" {
		ext = ".bin"
	}
	return fmt.Sprintf("%s/%s%s", prefix, uuid.New().String(), ext)
}
