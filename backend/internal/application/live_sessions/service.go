package live_sessions

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	domaincourses "lms-backend/internal/domain/courses"
	domain "lms-backend/internal/domain/live_sessions"
	"lms-backend/pkg/apperrors"
	"lms-backend/pkg/logger"

	"github.com/google/uuid"
)

// EnrollmentChecker verifies that a student is actively enrolled in a course.
type EnrollmentChecker interface {
	IsEnrolled(ctx context.Context, studentID, courseID uuid.UUID) (bool, error)
	ListStudentCourseIDs(ctx context.Context, studentID uuid.UUID) ([]uuid.UUID, error)
}

// NotificationEnqueuer enqueues notification jobs for enrolled students.
type NotificationEnqueuer interface {
	EnqueueSessionNotification(ctx context.Context, sessionID, courseID uuid.UUID, eventType string) error
}

// RecordingJobEnqueuer enqueues recording processing jobs.
type RecordingJobEnqueuer interface {
	EnqueueProcessRecording(ctx context.Context, sessionID uuid.UUID) error
}

// Service defines the live sessions use cases.
type Service interface {
	// ScheduleSession creates a new live session and notifies enrolled students.
	// Requirements: 16.1
	ScheduleSession(ctx context.Context, cmd ScheduleSessionCommand) (*LiveSessionResponse, error)

	// StartSession transitions the session to live and returns a teacher room token.
	// Requirements: 16.2
	StartSession(ctx context.Context, cmd StartSessionCommand) (*StartSessionResponse, error)

	// JoinSession verifies live status, records attendance, returns student room token.
	// Requirements: 16.3, 16.4
	JoinSession(ctx context.Context, cmd JoinSessionCommand) (*JoinSessionResponse, error)

	// EndSession transitions to ended, records attendee_count, enqueues recording job if needed.
	// Requirements: 16.5
	EndSession(ctx context.Context, cmd EndSessionCommand) (*LiveSessionResponse, error)

	// RescheduleOrCancelSession updates or cancels a session and notifies enrolled students.
	// Requirements: 16.6
	RescheduleOrCancelSession(ctx context.Context, cmd RescheduleOrCancelSessionCommand) (*LiveSessionResponse, error)

	// GetAttendance returns per-student attendance records for a session.
	// Requirements: 16.7
	GetAttendance(ctx context.Context, cmd GetAttendanceCommand) (*AttendanceResponse, error)

	// Teacher read-side session APIs.
	ListTeacherSessions(ctx context.Context, cmd ListTeacherSessionsCommand) (*LiveSessionListResponse, error)
	GetTeacherSession(ctx context.Context, cmd GetTeacherSessionCommand) (*LiveSessionResponse, error)

	// Student read-side session APIs.
	ListStudentSessions(ctx context.Context, cmd ListStudentSessionsCommand) (*LiveSessionListResponse, error)
	GetStudentSession(ctx context.Context, cmd GetStudentSessionCommand) (*LiveSessionResponse, error)
}

type service struct {
	sessionRepo    domain.LiveSessionRepository
	attendanceRepo domain.AttendanceRepository
	enrollments    EnrollmentChecker
	notifier       NotificationEnqueuer
	recorder       RecordingJobEnqueuer
	settings       TimezoneProvider
	courses        domaincourses.CourseRepository
}

type TimezoneProvider interface {
	DefaultTimezone(ctx context.Context) string
}

// NewService creates a new live sessions service.
func NewService(
	sessionRepo domain.LiveSessionRepository,
	attendanceRepo domain.AttendanceRepository,
	enrollments EnrollmentChecker,
	notifier NotificationEnqueuer,
	recorder RecordingJobEnqueuer,
	courses domaincourses.CourseRepository,
) Service {
	return &service{
		sessionRepo:    sessionRepo,
		attendanceRepo: attendanceRepo,
		enrollments:    enrollments,
		notifier:       notifier,
		recorder:       recorder,
		courses:        courses,
	}
}

func NewServiceWithTimezone(
	sessionRepo domain.LiveSessionRepository,
	attendanceRepo domain.AttendanceRepository,
	enrollments EnrollmentChecker,
	notifier NotificationEnqueuer,
	recorder RecordingJobEnqueuer,
	courses domaincourses.CourseRepository,
	settings TimezoneProvider,
) Service {
	svc := NewService(sessionRepo, attendanceRepo, enrollments, notifier, recorder, courses).(*service)
	svc.settings = settings
	return svc
}

// ScheduleSession creates a new live session and notifies enrolled students.
// Requirements: 16.1
func (s *service) ScheduleSession(ctx context.Context, cmd ScheduleSessionCommand) (*LiveSessionResponse, error) {
	course, err := s.courses.FindByID(ctx, cmd.CourseID)
	if err != nil || course == nil {
		return nil, apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}
	if !course.IsOwnedBy(cmd.TeacherID) {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "you do not own this course")
	}

	now := time.Now().UTC()
	session := &domain.LiveSession{
		ID:              uuid.New(),
		CourseID:        cmd.CourseID,
		TeacherID:       cmd.TeacherID,
		Title:           cmd.Title,
		ScheduledAt:     cmd.ScheduledAt,
		DurationMinutes: cmd.DurationMinutes,
		Status:          domain.SessionStatusScheduled,
		RecordSession:   cmd.RecordSession,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, apperrors.NewInternalError("SESSION_CREATE_FAILED", "failed to create live session")
	}

	// Notify enrolled students asynchronously
	if s.notifier != nil {
		if err := s.notifier.EnqueueSessionNotification(ctx, session.ID, session.CourseID, "session_scheduled"); err != nil {
			logger.Error(ctx, "Failed to enqueue session scheduled notification", "session_id", session.ID, "error", err)
			// Non-fatal: session was created successfully
		}
	}

	logger.Info(ctx, "Live session scheduled", "session_id", session.ID, "course_id", session.CourseID)
	return toResponse(session), nil
}

// StartSession transitions the session to live and returns a teacher-scoped room token.
// Requirements: 16.2
func (s *service) StartSession(ctx context.Context, cmd StartSessionCommand) (*StartSessionResponse, error) {
	session, err := s.sessionRepo.FindByID(ctx, cmd.SessionID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("SESSION_NOT_FOUND", "live session not found")
	}

	if session.TeacherID != cmd.TeacherID {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "you do not own this session")
	}

	if session.Status != domain.SessionStatusScheduled {
		return nil, apperrors.NewSimpleValidationError("INVALID_STATUS", fmt.Sprintf("cannot start a session in %s status", session.Status))
	}

	session.Status = domain.SessionStatusLive
	session.UpdatedAt = time.Now().UTC()

	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return nil, apperrors.NewInternalError("SESSION_UPDATE_FAILED", "failed to start live session")
	}

	// Generate a teacher-scoped room token
	roomToken, err := generateRoomToken("teacher", session.ID, cmd.TeacherID)
	if err != nil {
		return nil, apperrors.NewInternalError("TOKEN_GENERATION_FAILED", "failed to generate room token")
	}

	logger.Info(ctx, "Live session started", "session_id", session.ID)
	return &StartSessionResponse{
		Session:   *toResponse(session),
		RoomToken: roomToken,
	}, nil
}

// JoinSession verifies live status, records attendance, returns student-scoped room token.
// Requirements: 16.3, 16.4
func (s *service) JoinSession(ctx context.Context, cmd JoinSessionCommand) (*JoinSessionResponse, error) {
	session, err := s.sessionRepo.FindByID(ctx, cmd.SessionID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("SESSION_NOT_FOUND", "live session not found")
	}

	// Requirement 16.4: reject if not live
	if session.Status != domain.SessionStatusLive {
		return nil, apperrors.NewSimpleValidationError("SESSION_NOT_LIVE", "session is not currently live")
	}

	// Verify enrollment
	enrolled, err := s.enrollments.IsEnrolled(ctx, cmd.StudentID, session.CourseID)
	if err != nil {
		return nil, apperrors.NewInternalError("ENROLLMENT_CHECK_FAILED", "failed to verify enrollment")
	}
	if !enrolled {
		return nil, apperrors.NewForbiddenError("NOT_ENROLLED", "you are not enrolled in this course")
	}

	// Record attendance (idempotent — update joined_at if already exists)
	existing, _ := s.attendanceRepo.FindBySessionAndStudent(ctx, cmd.SessionID, cmd.StudentID)
	if existing == nil {
		attendance := &domain.Attendance{
			ID:        uuid.New(),
			SessionID: cmd.SessionID,
			StudentID: cmd.StudentID,
			JoinedAt:  time.Now().UTC(),
		}
		if err := s.attendanceRepo.Create(ctx, attendance); err != nil {
			return nil, apperrors.NewInternalError("ATTENDANCE_CREATE_FAILED", "failed to record attendance")
		}
	}

	// Generate a student-scoped room token
	roomToken, err := generateRoomToken("student", session.ID, cmd.StudentID)
	if err != nil {
		return nil, apperrors.NewInternalError("TOKEN_GENERATION_FAILED", "failed to generate room token")
	}

	logger.Info(ctx, "Student joined live session", "session_id", session.ID, "student_id", cmd.StudentID)
	return &JoinSessionResponse{
		SessionID: session.ID,
		RoomToken: roomToken,
	}, nil
}

// EndSession transitions to ended, records attendee_count, enqueues recording job if needed.
// Requirements: 16.5
func (s *service) EndSession(ctx context.Context, cmd EndSessionCommand) (*LiveSessionResponse, error) {
	session, err := s.sessionRepo.FindByID(ctx, cmd.SessionID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("SESSION_NOT_FOUND", "live session not found")
	}

	if session.TeacherID != cmd.TeacherID {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "you do not own this session")
	}

	if session.Status != domain.SessionStatusLive {
		return nil, apperrors.NewSimpleValidationError("INVALID_STATUS", fmt.Sprintf("cannot end a session in %s status", session.Status))
	}

	// Count attendees
	records, err := s.attendanceRepo.FindBySessionID(ctx, cmd.SessionID)
	if err != nil {
		logger.Error(ctx, "Failed to count attendees", "session_id", cmd.SessionID, "error", err)
	}

	session.Status = domain.SessionStatusEnded
	session.AttendeeCount = len(records)
	session.UpdatedAt = time.Now().UTC()

	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return nil, apperrors.NewInternalError("SESSION_UPDATE_FAILED", "failed to end live session")
	}

	// Enqueue recording processing job if needed
	if session.RecordSession && s.recorder != nil {
		if err := s.recorder.EnqueueProcessRecording(ctx, session.ID); err != nil {
			logger.Error(ctx, "Failed to enqueue recording job", "session_id", session.ID, "error", err)
			// Non-fatal
		}
	}

	logger.Info(ctx, "Live session ended", "session_id", session.ID, "attendee_count", session.AttendeeCount)
	return toResponse(session), nil
}

// RescheduleOrCancelSession updates or cancels a session and notifies enrolled students.
// Requirements: 16.6
func (s *service) RescheduleOrCancelSession(ctx context.Context, cmd RescheduleOrCancelSessionCommand) (*LiveSessionResponse, error) {
	session, err := s.sessionRepo.FindByID(ctx, cmd.SessionID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("SESSION_NOT_FOUND", "live session not found")
	}

	if session.TeacherID != cmd.TeacherID {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "you do not own this session")
	}

	if session.Status == domain.SessionStatusEnded || session.Status == domain.SessionStatusLive {
		return nil, apperrors.NewSimpleValidationError("INVALID_STATUS", fmt.Sprintf("cannot modify a session in %s status", session.Status))
	}

	eventType := "session_rescheduled"
	if cmd.Cancel {
		session.Status = domain.SessionStatusCancelled
		eventType = "session_cancelled"
	} else {
		if cmd.ScheduledAt != nil {
			session.ScheduledAt = *cmd.ScheduledAt
		}
		if cmd.DurationMinutes != nil {
			session.DurationMinutes = *cmd.DurationMinutes
		}
		if cmd.Title != nil {
			session.Title = *cmd.Title
		}
	}
	session.UpdatedAt = time.Now().UTC()

	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return nil, apperrors.NewInternalError("SESSION_UPDATE_FAILED", "failed to update live session")
	}

	// Notify enrolled students
	if s.notifier != nil {
		if err := s.notifier.EnqueueSessionNotification(ctx, session.ID, session.CourseID, eventType); err != nil {
			logger.Error(ctx, "Failed to enqueue session notification", "session_id", session.ID, "event", eventType, "error", err)
		}
	}

	logger.Info(ctx, "Live session updated", "session_id", session.ID, "event", eventType)
	return toResponse(session), nil
}

// GetAttendance returns per-student attendance records for a session.
// Requirements: 16.7
func (s *service) GetAttendance(ctx context.Context, cmd GetAttendanceCommand) (*AttendanceResponse, error) {
	session, err := s.sessionRepo.FindByID(ctx, cmd.SessionID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("SESSION_NOT_FOUND", "live session not found")
	}

	if session.TeacherID != cmd.TeacherID {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "you do not own this session")
	}

	records, err := s.attendanceRepo.FindBySessionID(ctx, cmd.SessionID)
	if err != nil {
		return nil, apperrors.NewInternalError("ATTENDANCE_QUERY_FAILED", "failed to retrieve attendance records")
	}

	attendanceRecords := make([]AttendanceRecord, 0, len(records))
	for _, r := range records {
		attendanceRecords = append(attendanceRecords, AttendanceRecord{
			StudentID:       r.StudentID,
			JoinedAt:        r.JoinedAt,
			LeftAt:          r.LeftAt,
			DurationMinutes: r.DurationMinutes,
		})
	}

	return &AttendanceResponse{
		SessionID: cmd.SessionID,
		Records:   attendanceRecords,
	}, nil
}

// ListTeacherSessions returns all sessions owned by the authenticated teacher.
func (s *service) ListTeacherSessions(ctx context.Context, cmd ListTeacherSessionsCommand) (*LiveSessionListResponse, error) {
	sessions, err := s.sessionRepo.FindByTeacherID(ctx, cmd.TeacherID)
	if err != nil {
		return nil, apperrors.NewInternalError("SESSION_QUERY_FAILED", "failed to retrieve live sessions")
	}

	response := make([]LiveSessionResponse, 0, len(sessions))
	for _, session := range sessions {
		response = append(response, *toResponse(session))
	}

	return &LiveSessionListResponse{Sessions: response}, nil
}

// GetTeacherSession returns a single teacher-owned session.
func (s *service) GetTeacherSession(ctx context.Context, cmd GetTeacherSessionCommand) (*LiveSessionResponse, error) {
	session, err := s.sessionRepo.FindByID(ctx, cmd.SessionID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("SESSION_NOT_FOUND", "live session not found")
	}

	if session.TeacherID != cmd.TeacherID {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "you do not own this session")
	}

	return toResponse(session), nil
}

// ListStudentSessions returns sessions across the student's enrolled courses.
func (s *service) ListStudentSessions(ctx context.Context, cmd ListStudentSessionsCommand) (*LiveSessionListResponse, error) {
	courseIDs, err := s.enrollments.ListStudentCourseIDs(ctx, cmd.StudentID)
	if err != nil {
		return nil, apperrors.NewInternalError("SESSION_QUERY_FAILED", "failed to retrieve enrolled courses")
	}

	sessions, err := s.sessionRepo.FindByCourseIDsBetween(ctx, courseIDs, cmd.From, cmd.To)
	if err != nil {
		return nil, apperrors.NewInternalError("SESSION_QUERY_FAILED", "failed to retrieve live sessions")
	}

	response := make([]LiveSessionResponse, 0, len(sessions))
	for _, session := range sessions {
		response = append(response, *s.toResponseWithToday(ctx, session))
	}

	return &LiveSessionListResponse{Sessions: response}, nil
}

// GetStudentSession returns a single session if the student is enrolled in its course.
func (s *service) GetStudentSession(ctx context.Context, cmd GetStudentSessionCommand) (*LiveSessionResponse, error) {
	session, err := s.sessionRepo.FindByID(ctx, cmd.SessionID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("SESSION_NOT_FOUND", "live session not found")
	}

	enrolled, err := s.enrollments.IsEnrolled(ctx, cmd.StudentID, session.CourseID)
	if err != nil {
		return nil, apperrors.NewInternalError("ENROLLMENT_CHECK_FAILED", "failed to verify enrollment")
	}
	if !enrolled {
		return nil, apperrors.NewForbiddenError("NOT_ENROLLED", "you are not enrolled in this course")
	}

	return toResponse(session), nil
}

// toResponse converts a domain LiveSession to a LiveSessionResponse.
func toResponse(s *domain.LiveSession) *LiveSessionResponse {
	return &LiveSessionResponse{
		ID:              s.ID,
		CourseID:        s.CourseID,
		TeacherID:       s.TeacherID,
		Title:           s.Title,
		ScheduledAt:     s.ScheduledAt,
		DurationMinutes: s.DurationMinutes,
		Status:          string(s.Status),
		RecordSession:   s.RecordSession,
		AttendeeCount:   s.AttendeeCount,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
	}
}

func (s *service) toResponseWithToday(ctx context.Context, session *domain.LiveSession) *LiveSessionResponse {
	response := toResponse(session)
	timezone := "UTC"
	if s.settings != nil {
		timezone = s.settings.DefaultTimezone(ctx)
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	scheduled := session.ScheduledAt.In(loc)
	response.IsToday = now.Year() == scheduled.Year() && now.YearDay() == scheduled.YearDay()
	return response
}

// generateRoomToken generates a scoped room token for a session participant.
// In production this would be a signed JWT or a third-party video provider token.
func generateRoomToken(role string, sessionID, userID uuid.UUID) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s_%s_%s_%s", role, sessionID, userID, hex.EncodeToString(b)), nil
}
