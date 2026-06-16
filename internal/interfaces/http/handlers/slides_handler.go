package handlers

import (
	"io"
	"net/http"
	"strconv"

	appslides "lms-backend/internal/application/slides"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

type SlidesHandler struct {
	service appslides.Service
}

func NewSlidesHandler(service appslides.Service) *SlidesHandler {
	return &SlidesHandler{service: service}
}

func (h *SlidesHandler) ListPublicSlides(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.ListPublicSlides(r.Context())
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

func (h *SlidesHandler) ListAdminSlides(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.ListAdminSlides(r.Context())
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

func (h *SlidesHandler) CreateSlide(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 6*1024*1024)
	if err := r.ParseMultipartForm(6 * 1024 * 1024); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_MULTIPART", "invalid multipart form"))
		return
	}
	file, header, err := r.FormFile("media")
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("MEDIA_REQUIRED", "media file is required"))
		return
	}
	defer file.Close()
	magic := make([]byte, 512)
	n, readErr := io.ReadFull(file, magic)
	if readErr != nil && readErr != io.ErrUnexpectedEOF {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_MEDIA", "could not read media file"))
		return
	}
	magic = magic[:n]

	result, err := h.service.CreateSlide(r.Context(), appslides.CreateSlideCommand{
		ActorID:    userID,
		Title:      r.FormValue("title"),
		Subtitle:   r.FormValue("subtitle"),
		LinkURL:    r.FormValue("link_url"),
		DurationMS: parseIntFormValue(r, "duration_ms", 5000),
		Position:   parseIntFormValue(r, "position", 0),
		FileName:   header.Filename,
		FileSize:   header.Size,
		MimeType:   header.Header.Get("Content-Type"),
		MagicBytes: magic,
		Reader:     appslides.RebuildReader(magic, file),
		IPAddress:  requestIP(r),
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusCreated, result)
}

func (h *SlidesHandler) UpdateSlide(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}
	slideID, err := uuid.Parse(r.PathValue("slideId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid slide ID"))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 6*1024*1024)
	if err := r.ParseMultipartForm(6 * 1024 * 1024); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_MULTIPART", "invalid multipart form"))
		return
	}
	var cmd = appslides.UpdateSlideCommand{
		ActorID:    userID,
		SlideID:    slideID,
		Title:      optionalFormString(r, "title"),
		Subtitle:   optionalFormString(r, "subtitle"),
		LinkURL:    optionalFormString(r, "link_url"),
		DurationMS: optionalFormInt(r, "duration_ms"),
		Position:   optionalFormInt(r, "position"),
		IsActive:   optionalFormBool(r, "is_active"),
		IPAddress:  requestIP(r),
	}
	file, header, err := r.FormFile("media")
	if err == nil {
		defer file.Close()
		magic := make([]byte, 512)
		n, readErr := io.ReadFull(file, magic)
		if readErr != nil && readErr != io.ErrUnexpectedEOF {
			writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_MEDIA", "could not read media file"))
			return
		}
		magic = magic[:n]
		cmd.FileName = header.Filename
		cmd.FileSize = header.Size
		cmd.MimeType = header.Header.Get("Content-Type")
		cmd.MagicBytes = magic
		cmd.Reader = appslides.RebuildReader(magic, file)
	}
	result, err := h.service.UpdateSlide(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

func (h *SlidesHandler) ReorderSlides(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}
	var req struct {
		Positions map[string]int `json:"positions"`
	}
	if err := decodeJSONBody(r, &req); err != nil {
		writeErrorResponse(w, err)
		return
	}
	positions, err := parseUUIDPositionMap(req.Positions)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	if err := h.service.ReorderSlides(r.Context(), appslides.ReorderSlidesCommand{ActorID: userID, Positions: positions, IPAddress: requestIP(r)}); err != nil {
		writeErrorResponse(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SlidesHandler) DeactivateSlide(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}
	slideID, err := uuid.Parse(r.PathValue("slideId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid slide ID"))
		return
	}
	if err := h.service.DeactivateSlide(r.Context(), appslides.DeactivateSlideCommand{ActorID: userID, SlideID: slideID, IPAddress: requestIP(r)}); err != nil {
		writeErrorResponse(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseIntFormValue(r *http.Request, key string, fallback int) int {
	value := r.FormValue(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func optionalFormString(r *http.Request, key string) *string {
	if _, ok := r.MultipartForm.Value[key]; !ok {
		return nil
	}
	value := r.FormValue(key)
	return &value
}

func optionalFormInt(r *http.Request, key string) *int {
	if _, ok := r.MultipartForm.Value[key]; !ok {
		return nil
	}
	value, err := strconv.Atoi(r.FormValue(key))
	if err != nil {
		return nil
	}
	return &value
}

func optionalFormBool(r *http.Request, key string) *bool {
	if _, ok := r.MultipartForm.Value[key]; !ok {
		return nil
	}
	value, err := strconv.ParseBool(r.FormValue(key))
	if err != nil {
		return nil
	}
	return &value
}
