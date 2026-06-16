package slides

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	domainslides "lms-backend/internal/domain/slides"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

const maxSlideMediaBytes int64 = 5 * 1024 * 1024

type StorageClient interface {
	PutObject(ctx context.Context, bucket, key string, r io.Reader, size int64, contentType string) error
	PresignGetURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error)
}

type AuditLogger interface {
	LogAction(ctx context.Context, actorID uuid.UUID, actorName, action, targetType string, targetID uuid.UUID, metadata map[string]interface{}, ipAddress string) error
}

type Service interface {
	CreateSlide(ctx context.Context, cmd CreateSlideCommand) (*SlideResponse, error)
	UpdateSlide(ctx context.Context, cmd UpdateSlideCommand) (*SlideResponse, error)
	ListAdminSlides(ctx context.Context) (*SlideListResponse, error)
	ListPublicSlides(ctx context.Context) (*SlideListResponse, error)
	ReorderSlides(ctx context.Context, cmd ReorderSlidesCommand) error
	DeactivateSlide(ctx context.Context, cmd DeactivateSlideCommand) error
}

type service struct {
	repo    domainslides.Repository
	storage StorageClient
	audit   AuditLogger
	bucket  string
}

func NewService(repo domainslides.Repository, storage StorageClient, audit AuditLogger, bucket string) Service {
	return &service{repo: repo, storage: storage, audit: audit, bucket: bucket}
}

func (s *service) CreateSlide(ctx context.Context, cmd CreateSlideCommand) (*SlideResponse, error) {
	mediaType, err := validateSlideMedia(cmd.FileSize, cmd.MimeType, cmd.MagicBytes)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.Title) == "" {
		return nil, apperrors.NewSimpleValidationError("TITLE_REQUIRED", "title is required")
	}
	if cmd.DurationMS <= 0 {
		cmd.DurationMS = 5000
	}

	now := time.Now().UTC()
	key := generatedSlideKey(cmd.FileName)
	if err := s.storage.PutObject(ctx, s.bucket, key, cmd.Reader, cmd.FileSize, mediaType); err != nil {
		return nil, apperrors.NewInternalError("MEDIA_UPLOAD_FAILED", "failed to store slide media")
	}

	slide := &domainslides.PromotionalSlide{
		ID:         uuid.New(),
		Title:      strings.TrimSpace(cmd.Title),
		Subtitle:   strings.TrimSpace(cmd.Subtitle),
		LinkURL:    strings.TrimSpace(cmd.LinkURL),
		MediaKey:   key,
		MediaType:  mediaType,
		DurationMS: cmd.DurationMS,
		Position:   cmd.Position,
		IsActive:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.repo.Create(ctx, slide); err != nil {
		return nil, err
	}
	s.log(ctx, cmd.ActorID, "slide_created", slide.ID, cmd.IPAddress)
	return s.withURL(ctx, slide)
}

func (s *service) UpdateSlide(ctx context.Context, cmd UpdateSlideCommand) (*SlideResponse, error) {
	slide, err := s.repo.FindByID(ctx, cmd.SlideID)
	if err != nil || slide == nil {
		return nil, apperrors.NewNotFoundError("SLIDE_NOT_FOUND", "slide not found")
	}
	if cmd.Title != nil {
		if strings.TrimSpace(*cmd.Title) == "" {
			return nil, apperrors.NewSimpleValidationError("TITLE_REQUIRED", "title is required")
		}
		slide.Title = strings.TrimSpace(*cmd.Title)
	}
	if cmd.Subtitle != nil {
		slide.Subtitle = strings.TrimSpace(*cmd.Subtitle)
	}
	if cmd.LinkURL != nil {
		slide.LinkURL = strings.TrimSpace(*cmd.LinkURL)
	}
	if cmd.DurationMS != nil {
		if *cmd.DurationMS <= 0 {
			return nil, apperrors.NewSimpleValidationError("INVALID_DURATION", "duration_ms must be positive")
		}
		slide.DurationMS = *cmd.DurationMS
	}
	if cmd.Position != nil {
		slide.Position = *cmd.Position
	}
	if cmd.IsActive != nil {
		slide.IsActive = *cmd.IsActive
	}
	if cmd.Reader != nil {
		mediaType, err := validateSlideMedia(cmd.FileSize, cmd.MimeType, cmd.MagicBytes)
		if err != nil {
			return nil, err
		}
		key := generatedSlideKey(cmd.FileName)
		if err := s.storage.PutObject(ctx, s.bucket, key, cmd.Reader, cmd.FileSize, mediaType); err != nil {
			return nil, apperrors.NewInternalError("MEDIA_UPLOAD_FAILED", "failed to store slide media")
		}
		slide.MediaKey = key
		slide.MediaType = mediaType
	}
	slide.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, slide); err != nil {
		return nil, err
	}
	s.log(ctx, cmd.ActorID, "slide_updated", slide.ID, cmd.IPAddress)
	return s.withURL(ctx, slide)
}

func (s *service) ListAdminSlides(ctx context.Context) (*SlideListResponse, error) {
	return s.list(ctx, false)
}

func (s *service) ListPublicSlides(ctx context.Context) (*SlideListResponse, error) {
	return s.list(ctx, true)
}

func (s *service) ReorderSlides(ctx context.Context, cmd ReorderSlidesCommand) error {
	if len(cmd.Positions) == 0 {
		return apperrors.NewSimpleValidationError("POSITIONS_REQUIRED", "positions are required")
	}
	if err := s.repo.Reorder(ctx, cmd.Positions); err != nil {
		return err
	}
	s.log(ctx, cmd.ActorID, "slides_reordered", uuid.Nil, cmd.IPAddress)
	return nil
}

func (s *service) DeactivateSlide(ctx context.Context, cmd DeactivateSlideCommand) error {
	slide, err := s.repo.FindByID(ctx, cmd.SlideID)
	if err != nil || slide == nil {
		return apperrors.NewNotFoundError("SLIDE_NOT_FOUND", "slide not found")
	}
	now := time.Now().UTC()
	slide.IsActive = false
	slide.DeactivatedAt = &now
	slide.UpdatedAt = now
	if err := s.repo.Update(ctx, slide); err != nil {
		return err
	}
	s.log(ctx, cmd.ActorID, "slide_deactivated", slide.ID, cmd.IPAddress)
	return nil
}

func (s *service) list(ctx context.Context, activeOnly bool) (*SlideListResponse, error) {
	slides, err := s.repo.List(ctx, activeOnly)
	if err != nil {
		return nil, err
	}
	response := make([]SlideResponse, 0, len(slides))
	for _, slide := range slides {
		item, err := s.withURL(ctx, slide)
		if err != nil {
			return nil, err
		}
		response = append(response, *item)
	}
	return &SlideListResponse{Slides: response}, nil
}

func (s *service) withURL(ctx context.Context, slide *domainslides.PromotionalSlide) (*SlideResponse, error) {
	url, err := s.storage.PresignGetURL(ctx, s.bucket, slide.MediaKey, 15*time.Minute)
	if err != nil {
		return nil, apperrors.NewInternalError("PRESIGN_FAILED", "failed to generate media URL")
	}
	response := toResponse(slide, url)
	return &response, nil
}

func validateSlideMedia(size int64, declared string, magic []byte) (string, error) {
	if size <= 0 || size > maxSlideMediaBytes {
		return "", apperrors.NewSimpleValidationError("INVALID_MEDIA_SIZE", "slide media must be greater than 0 and at most 5 MB")
	}
	detected := http.DetectContentType(magic)
	allowed := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
		"image/gif":  true,
	}
	if !allowed[detected] {
		return "", apperrors.NewSimpleValidationError("INVALID_MEDIA_TYPE", "slide media must be JPEG, PNG, WebP, or GIF")
	}
	if declared != "" && !strings.HasPrefix(declared, detected) && declared != "application/octet-stream" {
		return "", apperrors.NewSimpleValidationError("CONTENT_TYPE_MISMATCH", "declared content type does not match file contents")
	}
	return detected, nil
}

func generatedSlideKey(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" {
		ext = ".bin"
	}
	return fmt.Sprintf("promotional-slides/%s%s", uuid.New().String(), ext)
}

func (s *service) log(ctx context.Context, actorID uuid.UUID, action string, targetID uuid.UUID, ip string) {
	if s.audit == nil {
		return
	}
	_ = s.audit.LogAction(ctx, actorID, "", action, "promotional_slide", targetID, nil, ip)
}

func RebuildReader(magic []byte, remaining io.Reader) io.Reader {
	return io.MultiReader(bytes.NewReader(magic), remaining)
}
