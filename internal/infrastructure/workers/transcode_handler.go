package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lms-backend/internal/domain/courses"
	"lms-backend/internal/domain/notifications"
	"lms-backend/internal/infrastructure/rustfs"
	mp4tools "lms-backend/internal/infrastructure/video"
	"lms-backend/pkg/logger"

	"github.com/google/uuid"
)

// TranscodeDeps is the surface area the transcode handler needs from storage
// and the video repo. The cmd/server wiring passes concrete implementations;
// tests can substitute in-memory fakes.
type TranscodeDeps struct {
	Storage     rustfs.StorageClient
	VideoRepo   courses.VideoRepository
	VideoBucket string
}

// TranscodeHandler consumes transcode_video jobs, runs ffmpeg to produce an
// HLS bitrate ladder, uploads the result to RustFS, and updates the video
// row with the master manifest key.
//
// Failure handling: any error in the pipeline marks the video as "failed"
// in the DB so the polling UI can show an actionable message. We do not
// retry: ffmpeg failure usually means the source file is corrupt, and
// re-running will produce the same result. The teacher can re-upload.
type TranscodeHandler struct {
	deps TranscodeDeps
}

// NewTranscodeHandler builds a handler bound to the given deps.
func NewTranscodeHandler(deps TranscodeDeps) *TranscodeHandler {
	return &TranscodeHandler{deps: deps}
}

// transcodePayload is the JSON shape the uploader enqueues. Keeping the
// shape explicit (rather than `map[string]any`) lets us fail fast on bad
// payloads and gives the worker a typed entry point.
type transcodePayload struct {
	VideoID   string `json:"video_id"`
	RustFSKey string `json:"rustfs_key"`
}

// Handle implements JobHandler.
func (h *TranscodeHandler) Handle(ctx context.Context, job notifications.Job) error {
	var payload transcodePayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		logger.Error(ctx, "transcode job: bad payload", "error", err)
		return err
	}
	videoID, err := uuid.Parse(payload.VideoID)
	if err != nil {
		return fmt.Errorf("invalid video id: %w", err)
	}

	video, err := h.deps.VideoRepo.FindByID(ctx, videoID)
	if err != nil {
		return fmt.Errorf("load video: %w", err)
	}
	if video.Status != courses.VideoStatusProcessing {
		logger.Info(ctx, "skip transcode: video not in processing state", "video_id", videoID, "status", video.Status)
		return nil
	}

	logger.Info(ctx, "transcode: starting", "video_id", videoID, "rustfs_key", video.RustFSKey)

	if err := h.runTranscode(ctx, video); err != nil {
		logger.Error(ctx, "transcode: failed", "video_id", videoID, "error", err)
		// Mark as failed so the UI can surface a clear error to the teacher.
		// We swallow the DB error here — failing the handler a second time
		// is worse than a silently stuck row.
		video.Status = courses.VideoStatusFailed
		video.UpdatedAt = time.Now()
		_ = h.deps.VideoRepo.Update(ctx, video)
		return err
	}

	logger.Info(ctx, "transcode: complete", "video_id", videoID)
	return nil
}

// runTranscode does the actual work: download source to a temp file, invoke
// ffmpeg, upload outputs, update DB.
func (h *TranscodeHandler) runTranscode(ctx context.Context, video *courses.Video) error {
	sourcePath, cleanup, err := h.downloadSource(ctx, video)
	if err != nil {
		return err
	}
	defer cleanup()

	result, err := mp4tools.TranscodeToHLS(ctx, sourcePath, mp4tools.DefaultLadder)
	if err != nil {
		return fmt.Errorf("transcode: %w", err)
	}
	defer os.RemoveAll(result.OutputDir)

	if err := h.uploadOutputs(ctx, video, result); err != nil {
		return fmt.Errorf("upload outputs: %w", err)
	}

	now := time.Now()
	video.HLSManifestKey = fmt.Sprintf("videos/%s/hls/master.m3u8", video.RustFSKey)
	video.TranscodedAt = &now
	video.DurationSeconds = result.DurationSeconds
	video.Status = courses.VideoStatusReady
	video.UpdatedAt = time.Now()
	return h.deps.VideoRepo.Update(ctx, video)
}

// downloadSource pulls the original upload into a temp file. ffmpeg works
// better on local files than on streaming inputs because it can seek freely.
func (h *TranscodeHandler) downloadSource(ctx context.Context, video *courses.Video) (string, func(), error) {
	body, err := h.deps.Storage.GetObject(ctx, h.deps.VideoBucket, video.RustFSKey)
	if err != nil {
		return "", func() {}, fmt.Errorf("download source: %w", err)
	}
	defer body.Close()

	tmp, err := os.CreateTemp("", "transcode-source-*.mp4")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}
	if _, err := io.Copy(tmp, body); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return tmp.Name(), cleanup, nil
}

// uploadOutputs writes the master manifest, every rendition's playlist, and
// every .ts segment into the same bucket under hls/{videoKey}/. All writes
// happen sequentially because they are small (< 50 MB total typically) and
// ffmpeg just finished using the disk.
func (h *TranscodeHandler) uploadOutputs(ctx context.Context, video *courses.Video, result *mp4tools.TranscodeResult) error {
	prefix := fmt.Sprintf("videos/%s/hls", video.RustFSKey)
	masterKey := prefix + "/master.m3u8"

	if err := h.deps.Storage.PutObject(ctx, h.deps.VideoBucket, masterKey,
		strings.NewReader(string(result.MasterManifest)),
		int64(len(result.MasterManifest)),
		"application/vnd.apple.mpegurl",
	); err != nil {
		return fmt.Errorf("upload master manifest: %w", err)
	}

	for name, rend := range result.Renditions {
		playlistKey := fmt.Sprintf("%s/%s/playlist.m3u8", prefix, name)
		if err := h.deps.Storage.PutObject(ctx, h.deps.VideoBucket, playlistKey,
			strings.NewReader(string(rend.Playlist)),
			int64(len(rend.Playlist)),
			"application/vnd.apple.mpegurl",
		); err != nil {
			return fmt.Errorf("upload playlist %s: %w", name, err)
		}
		for _, seg := range rend.Segments {
			if err := h.uploadSegment(ctx, seg, prefix, name); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *TranscodeHandler) uploadSegment(ctx context.Context, seg mp4tools.SegmentFile, prefix, name string) error {
	f, err := os.Open(seg.Path)
	if err != nil {
		return fmt.Errorf("open segment: %w", err)
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat segment: %w", err)
	}
	key := fmt.Sprintf("%s/%s/%s", prefix, name, filepath.Base(seg.Path))
	if err := h.deps.Storage.PutObject(ctx, h.deps.VideoBucket, key, f, stat.Size(), "video/mp2t"); err != nil {
		return fmt.Errorf("upload segment %s: %w", key, err)
	}
	return nil
}

// Stub for filepath import shadowing inside workers package. Go's stdlib
// `path/filepath` is fine here, but linters sometimes flag the import if
// unused elsewhere in the file. The blank assignment keeps it.
var _ = errors.New
