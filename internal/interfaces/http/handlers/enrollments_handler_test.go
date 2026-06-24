package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"lms-backend/internal/application/enrollments"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// Mock service for testing
type mockEnrollmentsService struct {
	enrollFreeFunc            func(ctx context.Context, cmd enrollments.EnrollFreeCommand) (*enrollments.EnrollmentResponse, error)
	getStreamingSignedURLFunc func(ctx context.Context, cmd enrollments.GetStreamingSignedURLCommand) (*enrollments.StreamingSignedURLResponse, error)
	updateLessonProgressFunc  func(ctx context.Context, cmd enrollments.UpdateLessonProgressCommand) (*enrollments.LessonProgressResponse, error)
	getEnrollmentFunc         func(ctx context.Context, studentID, courseID uuid.UUID) (*enrollments.EnrollmentResponse, error)
}

func (m *mockEnrollmentsService) EnrollFree(ctx context.Context, cmd enrollments.EnrollFreeCommand) (*enrollments.EnrollmentResponse, error) {
	if m.enrollFreeFunc != nil {
		return m.enrollFreeFunc(ctx, cmd)
	}
	return nil, nil
}

func (m *mockEnrollmentsService) RevokeEnrollment(ctx context.Context, cmd enrollments.RevokeEnrollmentCommand) error {
	return nil
}

func (m *mockEnrollmentsService) GetEnrollment(ctx context.Context, studentID, courseID uuid.UUID) (*enrollments.EnrollmentResponse, error) {
	if m.getEnrollmentFunc != nil {
		return m.getEnrollmentFunc(ctx, studentID, courseID)
	}
	return nil, nil
}

func (m *mockEnrollmentsService) ListStudentEnrollments(ctx context.Context, studentID uuid.UUID, status string, page, limit int) ([]enrollments.EnrollmentResponse, int, error) {
	return nil, 0, nil
}

func (m *mockEnrollmentsService) UpdateLessonProgress(ctx context.Context, cmd enrollments.UpdateLessonProgressCommand) (*enrollments.LessonProgressResponse, error) {
	if m.updateLessonProgressFunc != nil {
		return m.updateLessonProgressFunc(ctx, cmd)
	}
	return nil, nil
}

func (m *mockEnrollmentsService) GetLessonProgress(ctx context.Context, enrollmentID, lessonID uuid.UUID) (*enrollments.LessonProgressResponse, error) {
	return nil, nil
}

func (m *mockEnrollmentsService) GetStreamingSignedURL(ctx context.Context, cmd enrollments.GetStreamingSignedURLCommand) (*enrollments.StreamingSignedURLResponse, error) {
	if m.getStreamingSignedURLFunc != nil {
		return m.getStreamingSignedURLFunc(ctx, cmd)
	}
	return nil, nil
}

func TestCreateEnrollment_Success(t *testing.T) {
	userID := uuid.New()
	courseID := uuid.New()
	enrollmentID := uuid.New()

	mockService := &mockEnrollmentsService{
		enrollFreeFunc: func(ctx context.Context, cmd enrollments.EnrollFreeCommand) (*enrollments.EnrollmentResponse, error) {
			if cmd.StudentID != userID {
				t.Errorf("expected student ID %s, got %s", userID, cmd.StudentID)
			}
			if cmd.CourseID != courseID {
				t.Errorf("expected course ID %s, got %s", courseID, cmd.CourseID)
			}
			return &enrollments.EnrollmentResponse{
				ID:             enrollmentID,
				StudentID:      userID,
				CourseID:       courseID,
				EnrollmentType: "free",
				Status:         "active",
				EnrolledAt:     time.Now(),
			}, nil
		},
	}

	handler := NewEnrollmentsHandler(mockService)

	reqBody := map[string]string{
		"course_id": courseID.String(),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/enrollments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Add user context
	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.CreateEnrollment(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response enrollments.EnrollmentResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.ID != enrollmentID {
		t.Errorf("expected enrollment ID %s, got %s", enrollmentID, response.ID)
	}
}

func TestCreateEnrollment_AlreadyEnrolled(t *testing.T) {
	userID := uuid.New()
	courseID := uuid.New()

	mockService := &mockEnrollmentsService{
		enrollFreeFunc: func(ctx context.Context, cmd enrollments.EnrollFreeCommand) (*enrollments.EnrollmentResponse, error) {
			return nil, apperrors.NewConflictError("ALREADY_ENROLLED", "you are already enrolled in this course")
		},
	}

	handler := NewEnrollmentsHandler(mockService)

	reqBody := map[string]string{
		"course_id": courseID.String(),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/enrollments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.CreateEnrollment(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, w.Code)
	}
}

func TestGetStreamingSignedURL_Success(t *testing.T) {
	userID := uuid.New()
	lessonID := uuid.New()
	signedURL := "https://example.com/signed-url"

	mockService := &mockEnrollmentsService{
		getStreamingSignedURLFunc: func(ctx context.Context, cmd enrollments.GetStreamingSignedURLCommand) (*enrollments.StreamingSignedURLResponse, error) {
			if cmd.UserID != userID {
				t.Errorf("expected user ID %s, got %s", userID, cmd.UserID)
			}
			if cmd.LessonID != lessonID {
				t.Errorf("expected lesson ID %s, got %s", lessonID, cmd.LessonID)
			}
			return &enrollments.StreamingSignedURLResponse{
				SignedURL: signedURL,
				ExpiresAt: time.Now().Add(2 * time.Hour),
			}, nil
		},
	}

	handler := NewEnrollmentsHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/v1/stream/lessons/"+lessonID.String()+"/signed-url", nil)
	req.SetPathValue("lessonId", lessonID.String())

	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetStreamingSignedURL(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response enrollments.StreamingSignedURLResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.SignedURL != signedURL {
		t.Errorf("expected signed URL %s, got %s", signedURL, response.SignedURL)
	}
}

func TestUpdateLessonProgress_Success(t *testing.T) {
	userID := uuid.New()
	courseID := uuid.New()
	lessonID := uuid.New()
	enrollmentID := uuid.New()
	progressID := uuid.New()

	mockService := &mockEnrollmentsService{
		getEnrollmentFunc: func(ctx context.Context, studentID, cID uuid.UUID) (*enrollments.EnrollmentResponse, error) {
			return &enrollments.EnrollmentResponse{
				ID:        enrollmentID,
				StudentID: studentID,
				CourseID:  cID,
			}, nil
		},
		updateLessonProgressFunc: func(ctx context.Context, cmd enrollments.UpdateLessonProgressCommand) (*enrollments.LessonProgressResponse, error) {
			if cmd.EnrollmentID != enrollmentID {
				t.Errorf("expected enrollment ID %s, got %s", enrollmentID, cmd.EnrollmentID)
			}
			if cmd.LessonID != lessonID {
				t.Errorf("expected lesson ID %s, got %s", lessonID, cmd.LessonID)
			}
			return &enrollments.LessonProgressResponse{
				ID:              progressID,
				EnrollmentID:    enrollmentID,
				LessonID:        lessonID,
				PositionSeconds: cmd.PositionSeconds,
				WatchedPercent:  cmd.WatchedPercent,
				Completed:       cmd.Completed,
			}, nil
		},
	}

	handler := NewEnrollmentsHandler(mockService)

	reqBody := map[string]interface{}{
		"position_seconds": 120,
		"watched_percent":  85.5,
		"completed":        true,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/enrollments/"+courseID.String()+"/lessons/"+lessonID.String()+"/progress", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("courseId", courseID.String())
	req.SetPathValue("lessonId", lessonID.String())

	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.UpdateLessonProgress(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response enrollments.LessonProgressResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.ID != progressID {
		t.Errorf("expected progress ID %s, got %s", progressID, response.ID)
	}
}
