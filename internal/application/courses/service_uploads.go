package courses

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"lms-backend/internal/domain/courses"
	"lms-backend/internal/domain/notifications"
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
	})
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

	if err := s.storage.PutObject(ctx, s.videoBucket, video.RustFSKey, cmd.Reader, cmd.FileSize, contentType); err != nil {
		logger.Error(ctx, "Video upload failed", "course_id", cmd.CourseID, "uploader_id", cmd.UploaderID, "file_name", cmd.FileName, "file_size", cmd.FileSize, "error", err)
		return nil, apperrors.NewInternalError("VIDEO_UPLOAD_FAILED", "failed to store video")
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
	})
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

func validateUpload(size, maxSize int64, declared string, magic []byte, allowed map[string]bool) (string, error) {
	if size <= 0 || size > maxSize {
		return "", apperrors.NewSimpleValidationError("INVALID_FILE_SIZE", fmt.Sprintf("file must be greater than 0 and at most %d bytes", maxSize))
	}
	detected := http.DetectContentType(magic)
	if strings.EqualFold(detected, "application/x-msdownload") || strings.Contains(detected, "executable") {
		return "", apperrors.NewSimpleValidationError("UNSAFE_FILE_TYPE", "executable uploads are not allowed")
	}
	if !allowed[detected] {
		return "", apperrors.NewSimpleValidationError("INVALID_FILE_TYPE", "file type is not allowed")
	}
	if declared != "" && declared != "application/octet-stream" && !strings.HasPrefix(declared, detected) {
		return "", apperrors.NewSimpleValidationError("CONTENT_TYPE_MISMATCH", "declared content type does not match file contents")
	}
	return detected, nil
}

func generatedUploadKey(prefix, fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" {
		ext = ".bin"
	}
	return fmt.Sprintf("%s/%s%s", prefix, uuid.New().String(), ext)
}
