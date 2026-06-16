package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"lms-backend/internal/application/assessments"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

// MockAssessmentService is a mock implementation of Service for testing
type MockAssessmentService struct {
	CreateQuizFunc                   func(ctx context.Context, cmd assessments.CreateQuizCommand) (*assessments.QuizResponse, error)
	GetTeacherQuizzesFunc            func(ctx context.Context, courseID, teacherID uuid.UUID) ([]assessments.QuizTeacherResponse, error)
	GetTeacherQuizFunc               func(ctx context.Context, quizID, teacherID uuid.UUID) (*assessments.QuizTeacherResponse, error)
	UpdateQuizFunc                   func(ctx context.Context, cmd assessments.UpdateQuizCommand) (*assessments.QuizTeacherResponse, error)
	DeleteQuizFunc                   func(ctx context.Context, quizID, teacherID uuid.UUID) error
	CreateQuestionFunc               func(ctx context.Context, cmd assessments.CreateStandaloneQuestionCommand) (*assessments.QuestionTeacherResponse, error)
	UpdateQuestionFunc               func(ctx context.Context, cmd assessments.UpdateQuestionCommand) (*assessments.QuestionTeacherResponse, error)
	DeleteQuestionFunc               func(ctx context.Context, questionID, teacherID uuid.UUID) error
	CreateAssignmentFunc             func(ctx context.Context, cmd assessments.CreateAssignmentCommand) (*assessments.AssignmentResponse, error)
	ListTeacherCourseAssignmentsFunc func(ctx context.Context, courseID, teacherID uuid.UUID) ([]assessments.AssignmentResponse, error)
	GetTeacherAssignmentFunc         func(ctx context.Context, assignmentID, teacherID uuid.UUID) (*assessments.AssignmentResponse, error)
	UpdateAssignmentFunc             func(ctx context.Context, cmd assessments.UpdateAssignmentCommand) (*assessments.AssignmentResponse, error)
	StartAttemptFunc                 func(ctx context.Context, cmd assessments.StartAttemptCommand) (*assessments.QuizAttemptResponse, error)
	SubmitAttemptFunc                func(ctx context.Context, cmd assessments.SubmitAttemptCommand) (*assessments.SubmitAttemptResponse, error)
	SubmitAssignmentFunc             func(ctx context.Context, cmd assessments.SubmitAssignmentCommand) (*assessments.AssignmentSubmissionResponse, error)
	GradeSubmissionFunc              func(ctx context.Context, cmd assessments.GradeSubmissionCommand) (*assessments.SubmissionGradeResponse, error)
	AutoSubmitAttemptFunc            func(ctx context.Context, attemptID uuid.UUID) (*assessments.SubmitAttemptResponse, error)
}

func (m *MockAssessmentService) CreateQuiz(ctx context.Context, cmd assessments.CreateQuizCommand) (*assessments.QuizResponse, error) {
	if m.CreateQuizFunc != nil {
		return m.CreateQuizFunc(ctx, cmd)
	}
	return nil, nil
}

func (m *MockAssessmentService) GetTeacherQuizzes(ctx context.Context, courseID, teacherID uuid.UUID) ([]assessments.QuizTeacherResponse, error) {
	if m.GetTeacherQuizzesFunc != nil {
		return m.GetTeacherQuizzesFunc(ctx, courseID, teacherID)
	}
	return nil, nil
}

func (m *MockAssessmentService) GetTeacherQuiz(ctx context.Context, quizID, teacherID uuid.UUID) (*assessments.QuizTeacherResponse, error) {
	if m.GetTeacherQuizFunc != nil {
		return m.GetTeacherQuizFunc(ctx, quizID, teacherID)
	}
	return nil, nil
}

func (m *MockAssessmentService) UpdateQuiz(ctx context.Context, cmd assessments.UpdateQuizCommand) (*assessments.QuizTeacherResponse, error) {
	if m.UpdateQuizFunc != nil {
		return m.UpdateQuizFunc(ctx, cmd)
	}
	return nil, nil
}

func (m *MockAssessmentService) DeleteQuiz(ctx context.Context, quizID, teacherID uuid.UUID) error {
	if m.DeleteQuizFunc != nil {
		return m.DeleteQuizFunc(ctx, quizID, teacherID)
	}
	return nil
}

func (m *MockAssessmentService) CreateQuestion(ctx context.Context, cmd assessments.CreateStandaloneQuestionCommand) (*assessments.QuestionTeacherResponse, error) {
	if m.CreateQuestionFunc != nil {
		return m.CreateQuestionFunc(ctx, cmd)
	}
	return nil, nil
}

func (m *MockAssessmentService) UpdateQuestion(ctx context.Context, cmd assessments.UpdateQuestionCommand) (*assessments.QuestionTeacherResponse, error) {
	if m.UpdateQuestionFunc != nil {
		return m.UpdateQuestionFunc(ctx, cmd)
	}
	return nil, nil
}

func (m *MockAssessmentService) DeleteQuestion(ctx context.Context, questionID, teacherID uuid.UUID) error {
	if m.DeleteQuestionFunc != nil {
		return m.DeleteQuestionFunc(ctx, questionID, teacherID)
	}
	return nil
}

func (m *MockAssessmentService) CreateAssignment(ctx context.Context, cmd assessments.CreateAssignmentCommand) (*assessments.AssignmentResponse, error) {
	if m.CreateAssignmentFunc != nil {
		return m.CreateAssignmentFunc(ctx, cmd)
	}
	return nil, nil
}

func (m *MockAssessmentService) ListTeacherCourseAssignments(ctx context.Context, courseID, teacherID uuid.UUID) ([]assessments.AssignmentResponse, error) {
	if m.ListTeacherCourseAssignmentsFunc != nil {
		return m.ListTeacherCourseAssignmentsFunc(ctx, courseID, teacherID)
	}
	return nil, nil
}

func (m *MockAssessmentService) GetTeacherAssignment(ctx context.Context, assignmentID, teacherID uuid.UUID) (*assessments.AssignmentResponse, error) {
	if m.GetTeacherAssignmentFunc != nil {
		return m.GetTeacherAssignmentFunc(ctx, assignmentID, teacherID)
	}
	return nil, nil
}

func (m *MockAssessmentService) UpdateAssignment(ctx context.Context, cmd assessments.UpdateAssignmentCommand) (*assessments.AssignmentResponse, error) {
	if m.UpdateAssignmentFunc != nil {
		return m.UpdateAssignmentFunc(ctx, cmd)
	}
	return nil, nil
}

func (m *MockAssessmentService) StartAttempt(ctx context.Context, cmd assessments.StartAttemptCommand) (*assessments.QuizAttemptResponse, error) {
	if m.StartAttemptFunc != nil {
		return m.StartAttemptFunc(ctx, cmd)
	}
	return nil, nil
}

func (m *MockAssessmentService) SubmitAttempt(ctx context.Context, cmd assessments.SubmitAttemptCommand) (*assessments.SubmitAttemptResponse, error) {
	if m.SubmitAttemptFunc != nil {
		return m.SubmitAttemptFunc(ctx, cmd)
	}
	return nil, nil
}

func (m *MockAssessmentService) SubmitAssignment(ctx context.Context, cmd assessments.SubmitAssignmentCommand) (*assessments.AssignmentSubmissionResponse, error) {
	if m.SubmitAssignmentFunc != nil {
		return m.SubmitAssignmentFunc(ctx, cmd)
	}
	return nil, nil
}

func (m *MockAssessmentService) GradeSubmission(ctx context.Context, cmd assessments.GradeSubmissionCommand) (*assessments.SubmissionGradeResponse, error) {
	if m.GradeSubmissionFunc != nil {
		return m.GradeSubmissionFunc(ctx, cmd)
	}
	return nil, nil
}

func (m *MockAssessmentService) AutoSubmitAttempt(ctx context.Context, attemptID uuid.UUID) (*assessments.SubmitAttemptResponse, error) {
	if m.AutoSubmitAttemptFunc != nil {
		return m.AutoSubmitAttemptFunc(ctx, attemptID)
	}
	return nil, nil
}

func (m *MockAssessmentService) ListStudentQuizzes(ctx context.Context, studentID uuid.UUID) ([]assessments.StudentQuizSummaryResponse, error) {
	return nil, nil
}

func (m *MockAssessmentService) GetStudentQuizDetail(ctx context.Context, quizID, studentID uuid.UUID) (*assessments.StudentQuizDetailResponse, error) {
	return nil, nil
}

func (m *MockAssessmentService) GetStudentQuizAttemptResult(ctx context.Context, quizID, attemptID, studentID uuid.UUID) (*assessments.StudentAttemptResult, error) {
	return nil, nil
}

func (m *MockAssessmentService) ListStudentAssignments(ctx context.Context, studentID uuid.UUID) ([]assessments.StudentAssignmentDetailResponse, error) {
	return nil, nil
}

func (m *MockAssessmentService) GetStudentAssignmentDetail(ctx context.Context, assignmentID, studentID uuid.UUID) (*assessments.StudentAssignmentDetailResponse, error) {
	return nil, nil
}

func (m *MockAssessmentService) ListTeacherAssignmentSubmissions(ctx context.Context, assignmentID, teacherID uuid.UUID, page, limit int) (*assessments.TeacherAssignmentSubmissionListResponse, error) {
	return nil, nil
}

func (m *MockAssessmentService) GetTeacherAssignmentSubmission(ctx context.Context, assignmentID, submissionID, teacherID uuid.UUID) (*assessments.AssignmentSubmissionResponse, error) {
	return nil, nil
}

func TestCreateQuiz(t *testing.T) {
	courseID := uuid.New()
	teacherID := uuid.New()
	quizID := uuid.New()

	mockService := &MockAssessmentService{
		CreateQuizFunc: func(ctx context.Context, cmd assessments.CreateQuizCommand) (*assessments.QuizResponse, error) {
			if cmd.CourseID != courseID {
				t.Errorf("expected course ID %v, got %v", courseID, cmd.CourseID)
			}
			if cmd.TeacherID != teacherID {
				t.Errorf("expected teacher ID %v, got %v", teacherID, cmd.TeacherID)
			}

			return &assessments.QuizResponse{
				ID:                         quizID,
				CourseID:                   courseID,
				Title:                      cmd.Title,
				TimeLimitSeconds:           cmd.TimeLimitSeconds,
				MaxAttempts:                cmd.MaxAttempts,
				PassingScorePercent:        cmd.PassingScorePercent,
				ShuffleQuestions:           cmd.ShuffleQuestions,
				ShowAnswersAfterSubmission: cmd.ShowAnswersAfterSubmission,
				CreatedAt:                  time.Now(),
			}, nil
		},
	}

	handler := NewAssessmentsHandler(mockService)

	reqBody := map[string]interface{}{
		"title":                         "Test Quiz",
		"time_limit_seconds":            1800,
		"max_attempts":                  3,
		"passing_score_percent":         60.0,
		"shuffle_questions":             true,
		"show_answers_after_submission": true,
		"questions": []map[string]interface{}{
			{
				"body":     "What is 2+2?",
				"type":     "single",
				"position": 1,
				"options": []map[string]interface{}{
					{"body": "3", "is_correct": false, "position": 1},
					{"body": "4", "is_correct": true, "position": 2},
				},
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/teacher/courses/"+courseID.String()+"/quizzes", bytes.NewReader(body))
	req.SetPathValue("courseId", courseID.String())

	ctx := context.WithValue(req.Context(), "user_id", teacherID)
	ctx = context.WithValue(ctx, "role", "teacher")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.CreateQuiz(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response assessments.QuizResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.ID != quizID {
		t.Errorf("expected quiz ID %v, got %v", quizID, response.ID)
	}
}

func TestStartQuizAttempt(t *testing.T) {
	quizID := uuid.New()
	studentID := uuid.New()
	attemptID := uuid.New()

	mockService := &MockAssessmentService{
		StartAttemptFunc: func(ctx context.Context, cmd assessments.StartAttemptCommand) (*assessments.QuizAttemptResponse, error) {
			if cmd.QuizID != quizID {
				t.Errorf("expected quiz ID %v, got %v", quizID, cmd.QuizID)
			}
			if cmd.StudentID != studentID {
				t.Errorf("expected student ID %v, got %v", studentID, cmd.StudentID)
			}

			return &assessments.QuizAttemptResponse{
				ID:        attemptID,
				QuizID:    quizID,
				StartedAt: time.Now(),
			}, nil
		},
	}

	handler := NewAssessmentsHandler(mockService)

	req := httptest.NewRequest(http.MethodPost, "/v1/quizzes/"+quizID.String()+"/attempts", nil)
	req.SetPathValue("quizId", quizID.String())

	ctx := context.WithValue(req.Context(), "user_id", studentID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.StartQuizAttempt(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response assessments.QuizAttemptResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.ID != attemptID {
		t.Errorf("expected attempt ID %v, got %v", attemptID, response.ID)
	}
}

func TestCreateAssignment(t *testing.T) {
	courseID := uuid.New()
	teacherID := uuid.New()
	assignmentID := uuid.New()

	mockService := &MockAssessmentService{
		CreateAssignmentFunc: func(ctx context.Context, cmd assessments.CreateAssignmentCommand) (*assessments.AssignmentResponse, error) {
			if cmd.CourseID != courseID {
				t.Errorf("expected course ID %v, got %v", courseID, cmd.CourseID)
			}
			if cmd.TeacherID != teacherID {
				t.Errorf("expected teacher ID %v, got %v", teacherID, cmd.TeacherID)
			}

			return &assessments.AssignmentResponse{
				ID:                  assignmentID,
				CourseID:            courseID,
				Title:               cmd.Title,
				Description:         cmd.Description,
				DueAt:               time.Now().Add(7 * 24 * time.Hour),
				SubmissionType:      cmd.SubmissionType,
				MaxFileSizeMB:       cmd.MaxFileSizeMB,
				AllowLateSubmission: cmd.AllowLateSubmission,
				TotalMarks:          cmd.TotalMarks,
				CreatedAt:           time.Now(),
			}, nil
		},
	}

	handler := NewAssessmentsHandler(mockService)

	reqBody := map[string]interface{}{
		"title":                 "Test Assignment",
		"description":           "Complete the assignment",
		"due_at":                time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339),
		"submission_type":       "both",
		"max_file_size_mb":      50,
		"allow_late_submission": true,
		"total_marks":           100.0,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/teacher/courses/"+courseID.String()+"/assignments", bytes.NewReader(body))
	req.SetPathValue("courseId", courseID.String())

	ctx := context.WithValue(req.Context(), "user_id", teacherID)
	ctx = context.WithValue(ctx, "role", "teacher")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.CreateAssignment(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response assessments.AssignmentResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.ID != assignmentID {
		t.Errorf("expected assignment ID %v, got %v", assignmentID, response.ID)
	}
}
