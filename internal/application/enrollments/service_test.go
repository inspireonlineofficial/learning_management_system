package enrollments

import (
	"context"
	"testing"
	"time"

	"lms-backend/internal/domain/auth"
	"lms-backend/internal/domain/courses"
	"lms-backend/internal/domain/enrollments"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// Mock repositories for testing

type mockEnrollmentRepo struct {
	enrollments map[string]*enrollments.Enrollment
}

func newMockEnrollmentRepo() *mockEnrollmentRepo {
	return &mockEnrollmentRepo{
		enrollments: make(map[string]*enrollments.Enrollment),
	}
}

func (m *mockEnrollmentRepo) Create(ctx context.Context, enrollment *enrollments.Enrollment) error {
	key := enrollment.StudentID.String() + ":" + enrollment.CourseID.String()
	if _, exists := m.enrollments[key]; exists {
		return &apperrors.AppError{Code: "ALREADY_ENROLLED"}
	}
	m.enrollments[key] = enrollment
	return nil
}

func (m *mockEnrollmentRepo) FindByID(ctx context.Context, id uuid.UUID) (*enrollments.Enrollment, error) {
	for _, e := range m.enrollments {
		if e.ID == id {
			return e, nil
		}
	}
	return nil, &apperrors.AppError{Code: "NOT_FOUND"}
}

func (m *mockEnrollmentRepo) FindByStudentAndCourse(ctx context.Context, studentID, courseID uuid.UUID) (*enrollments.Enrollment, error) {
	key := studentID.String() + ":" + courseID.String()
	if e, exists := m.enrollments[key]; exists {
		return e, nil
	}
	return nil, &apperrors.AppError{Code: "NOT_FOUND"}
}

func (m *mockEnrollmentRepo) FindByStudentID(ctx context.Context, studentID uuid.UUID, page, limit int) ([]*enrollments.Enrollment, int, error) {
	var result []*enrollments.Enrollment
	for _, e := range m.enrollments {
		if e.StudentID == studentID {
			result = append(result, e)
		}
	}
	return result, len(result), nil
}

func (m *mockEnrollmentRepo) FindByCourseID(ctx context.Context, courseID uuid.UUID, page, limit int) ([]*enrollments.Enrollment, int, error) {
	var result []*enrollments.Enrollment
	for _, e := range m.enrollments {
		if e.CourseID == courseID {
			result = append(result, e)
		}
	}
	return result, len(result), nil
}

func (m *mockEnrollmentRepo) Update(ctx context.Context, enrollment *enrollments.Enrollment) error {
	key := enrollment.StudentID.String() + ":" + enrollment.CourseID.String()
	if _, exists := m.enrollments[key]; !exists {
		return &apperrors.AppError{Code: "NOT_FOUND"}
	}
	m.enrollments[key] = enrollment
	return nil
}

func (m *mockEnrollmentRepo) UpdateProgressPercent(ctx context.Context, enrollmentID uuid.UUID, progressPercent float64) error {
	for _, e := range m.enrollments {
		if e.ID == enrollmentID {
			e.ProgressPercent = progressPercent
			return nil
		}
	}
	return &apperrors.AppError{Code: "NOT_FOUND"}
}

func (m *mockEnrollmentRepo) RecalculateProgressPercent(ctx context.Context, enrollmentID uuid.UUID) error {
	// Mock implementation - just set to 50% for testing
	for _, e := range m.enrollments {
		if e.ID == enrollmentID {
			e.ProgressPercent = 50.0
			return nil
		}
	}
	return &apperrors.AppError{Code: "NOT_FOUND"}
}

func (m *mockEnrollmentRepo) CountTotalLessons(ctx context.Context, courseID uuid.UUID) (int, error) {
	// Mock implementation - return 10 lessons
	return 10, nil
}

func (m *mockEnrollmentRepo) Exists(ctx context.Context, studentID, courseID uuid.UUID) (bool, error) {
	key := studentID.String() + ":" + courseID.String()
	_, exists := m.enrollments[key]
	return exists, nil
}

type mockLessonProgressRepo struct {
	progress map[string]*enrollments.LessonProgress
}

func newMockLessonProgressRepo() *mockLessonProgressRepo {
	return &mockLessonProgressRepo{
		progress: make(map[string]*enrollments.LessonProgress),
	}
}

func (m *mockLessonProgressRepo) Upsert(ctx context.Context, progress *enrollments.LessonProgress) error {
	key := progress.EnrollmentID.String() + ":" + progress.LessonID.String()
	m.progress[key] = progress
	return nil
}

func (m *mockLessonProgressRepo) FindByID(ctx context.Context, id uuid.UUID) (*enrollments.LessonProgress, error) {
	for _, p := range m.progress {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, &apperrors.AppError{Code: "NOT_FOUND"}
}

func (m *mockLessonProgressRepo) FindByEnrollmentAndLesson(ctx context.Context, enrollmentID, lessonID uuid.UUID) (*enrollments.LessonProgress, error) {
	key := enrollmentID.String() + ":" + lessonID.String()
	if p, exists := m.progress[key]; exists {
		return p, nil
	}
	return nil, &apperrors.AppError{Code: "NOT_FOUND"}
}

func (m *mockLessonProgressRepo) FindByEnrollmentID(ctx context.Context, enrollmentID uuid.UUID) ([]*enrollments.LessonProgress, error) {
	var result []*enrollments.LessonProgress
	for _, p := range m.progress {
		if p.EnrollmentID == enrollmentID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockLessonProgressRepo) CountCompletedLessons(ctx context.Context, enrollmentID uuid.UUID) (int, error) {
	count := 0
	for _, p := range m.progress {
		if p.EnrollmentID == enrollmentID && p.Completed {
			count++
		}
	}
	return count, nil
}

type mockCourseRepo struct {
	courses map[uuid.UUID]*courses.Course
}

func newMockCourseRepo() *mockCourseRepo {
	return &mockCourseRepo{
		courses: make(map[uuid.UUID]*courses.Course),
	}
}

func (m *mockCourseRepo) FindByID(ctx context.Context, id uuid.UUID) (*courses.Course, error) {
	if c, exists := m.courses[id]; exists {
		return c, nil
	}
	return nil, &apperrors.AppError{Code: "NOT_FOUND"}
}

func (m *mockCourseRepo) Create(ctx context.Context, course *courses.Course) error {
	m.courses[course.ID] = course
	return nil
}

func (m *mockCourseRepo) FindBySlug(ctx context.Context, slug string) (*courses.Course, error) {
	return nil, &apperrors.AppError{Code: "NOT_FOUND"}
}

func (m *mockCourseRepo) FindByTeacherID(ctx context.Context, teacherID uuid.UUID, page, limit int) ([]*courses.Course, int, error) {
	return nil, 0, nil
}

func (m *mockCourseRepo) Update(ctx context.Context, course *courses.Course) error {
	return nil
}

func (m *mockCourseRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockCourseRepo) List(ctx context.Context, filters courses.CourseFilters, page, limit int) ([]*courses.Course, int, error) {
	return nil, 0, nil
}

func (m *mockCourseRepo) CountPublishedLessons(ctx context.Context, courseID uuid.UUID) (int, error) {
	return 0, nil
}

type mockUserRepo struct {
	users map[uuid.UUID]*auth.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users: make(map[uuid.UUID]*auth.User),
	}
}

func (m *mockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*auth.User, error) {
	if u, exists := m.users[id]; exists {
		return u, nil
	}
	return nil, &apperrors.AppError{Code: "NOT_FOUND"}
}

func (m *mockUserRepo) Create(ctx context.Context, u *auth.User) error {
	m.users[u.ID] = u
	return nil
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*auth.User, error) {
	return nil, &apperrors.AppError{Code: "NOT_FOUND"}
}

func (m *mockUserRepo) FindByUsername(ctx context.Context, username string) (*auth.User, error) {
	return nil, &apperrors.AppError{Code: "NOT_FOUND"}
}

func (m *mockUserRepo) Update(ctx context.Context, u *auth.User) error {
	return nil
}

func (m *mockUserRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return nil
}

type mockSigningKeyStore struct{}

func (m *mockSigningKeyStore) InvalidateUserSigningKey(ctx context.Context, userID uuid.UUID, courseID uuid.UUID) error {
	return nil
}

type mockLessonRepo struct {
	lessons map[uuid.UUID]*courses.Lesson
}

func newMockLessonRepo() *mockLessonRepo {
	return &mockLessonRepo{
		lessons: make(map[uuid.UUID]*courses.Lesson),
	}
}

func (m *mockLessonRepo) FindByID(ctx context.Context, id uuid.UUID) (*courses.Lesson, error) {
	if l, exists := m.lessons[id]; exists {
		return l, nil
	}
	return nil, &apperrors.AppError{Code: "NOT_FOUND"}
}

func (m *mockLessonRepo) Create(ctx context.Context, lesson *courses.Lesson) error {
	m.lessons[lesson.ID] = lesson
	return nil
}

func (m *mockLessonRepo) FindByChapterID(ctx context.Context, chapterID uuid.UUID) ([]*courses.Lesson, error) {
	return nil, nil
}

func (m *mockLessonRepo) Update(ctx context.Context, lesson *courses.Lesson) error {
	return nil
}

func (m *mockLessonRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockLessonRepo) Reorder(ctx context.Context, chapterID uuid.UUID, positions map[uuid.UUID]int) error {
	return nil
}

type mockVideoRepo struct {
	videos map[uuid.UUID]*courses.Video
}

func newMockVideoRepo() *mockVideoRepo {
	return &mockVideoRepo{
		videos: make(map[uuid.UUID]*courses.Video),
	}
}

func (m *mockVideoRepo) FindByID(ctx context.Context, id uuid.UUID) (*courses.Video, error) {
	if v, exists := m.videos[id]; exists {
		return v, nil
	}
	return nil, &apperrors.AppError{Code: "NOT_FOUND"}
}

func (m *mockVideoRepo) Create(ctx context.Context, video *courses.Video) error {
	m.videos[video.ID] = video
	return nil
}

func (m *mockVideoRepo) Update(ctx context.Context, video *courses.Video) error {
	return nil
}

type mockStorageClient struct {
	presignedURLs map[string]string
}

func newMockStorageClient() *mockStorageClient {
	return &mockStorageClient{
		presignedURLs: make(map[string]string),
	}
}

func (m *mockStorageClient) PresignGetURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	url := "https://example.com/presigned/" + key + "?expires=" + ttl.String()
	m.presignedURLs[key] = url
	return url, nil
}

// Test EnrollFree use case

func TestEnrollFree_Success(t *testing.T) {
	enrollmentRepo := newMockEnrollmentRepo()
	lessonProgressRepo := newMockLessonProgressRepo()
	courseRepo := newMockCourseRepo()
	lessonRepo := newMockLessonRepo()
	videoRepo := newMockVideoRepo()
	userRepo := newMockUserRepo()
	signingKeyStore := &mockSigningKeyStore{}
	storageClient := newMockStorageClient()

	service := NewService(enrollmentRepo, lessonProgressRepo, courseRepo, lessonRepo, videoRepo, userRepo, signingKeyStore, storageClient, "videos")

	// Setup test data
	studentID := uuid.New()
	courseID := uuid.New()

	user := &auth.User{
		ID:              studentID,
		Role:            "student",
		ProfileComplete: true,
	}
	userRepo.users[studentID] = user

	course := &courses.Course{
		ID:        courseID,
		PriceType: courses.PriceTypeFree,
		Status:    courses.CourseStatusPublished,
	}
	courseRepo.courses[courseID] = course

	// Execute
	cmd := EnrollFreeCommand{
		StudentID: studentID,
		CourseID:  courseID,
	}

	result, err := service.EnrollFree(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.StudentID != studentID {
		t.Errorf("Expected student ID %v, got %v", studentID, result.StudentID)
	}

	if result.CourseID != courseID {
		t.Errorf("Expected course ID %v, got %v", courseID, result.CourseID)
	}

	if result.Status != string(enrollments.EnrollmentStatusActive) {
		t.Errorf("Expected status active, got %v", result.Status)
	}
}

func TestEnrollFree_ProfileIncomplete(t *testing.T) {
	enrollmentRepo := newMockEnrollmentRepo()
	courseRepo := newMockCourseRepo()
	lessonRepo := newMockLessonRepo()
	videoRepo := newMockVideoRepo()
	userRepo := newMockUserRepo()
	signingKeyStore := &mockSigningKeyStore{}
	storageClient := newMockStorageClient()

	lessonProgressRepo := newMockLessonProgressRepo()

	service := NewService(enrollmentRepo, lessonProgressRepo, courseRepo, lessonRepo, videoRepo, userRepo, signingKeyStore, storageClient, "videos")

	// Setup test data
	studentID := uuid.New()
	courseID := uuid.New()

	user := &auth.User{
		ID:              studentID,
		Role:            "student",
		ProfileComplete: false, // Profile not complete
	}
	userRepo.users[studentID] = user

	course := &courses.Course{
		ID:        courseID,
		PriceType: courses.PriceTypeFree,
		Status:    courses.CourseStatusPublished,
	}
	courseRepo.courses[courseID] = course

	// Execute
	cmd := EnrollFreeCommand{
		StudentID: studentID,
		CourseID:  courseID,
	}

	_, err := service.EnrollFree(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("Expected error for incomplete profile, got nil")
	}

	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("Expected AppError, got %T", err)
	}

	if appErr.Code != "PROFILE_INCOMPLETE" {
		t.Errorf("Expected error code PROFILE_INCOMPLETE, got %v", appErr.Code)
	}
}

func TestEnrollFree_PaidCourse(t *testing.T) {
	enrollmentRepo := newMockEnrollmentRepo()
	courseRepo := newMockCourseRepo()
	lessonRepo := newMockLessonRepo()
	videoRepo := newMockVideoRepo()
	userRepo := newMockUserRepo()
	signingKeyStore := &mockSigningKeyStore{}
	storageClient := newMockStorageClient()

	lessonProgressRepo := newMockLessonProgressRepo()

	service := NewService(enrollmentRepo, lessonProgressRepo, courseRepo, lessonRepo, videoRepo, userRepo, signingKeyStore, storageClient, "videos")

	// Setup test data
	studentID := uuid.New()
	courseID := uuid.New()

	user := &auth.User{
		ID:              studentID,
		Role:            "student",
		ProfileComplete: true,
	}
	userRepo.users[studentID] = user

	course := &courses.Course{
		ID:        courseID,
		PriceType: courses.PriceTypePaid, // Paid course
		Status:    courses.CourseStatusPublished,
	}
	courseRepo.courses[courseID] = course

	// Execute
	cmd := EnrollFreeCommand{
		StudentID: studentID,
		CourseID:  courseID,
	}

	_, err := service.EnrollFree(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("Expected error for paid course, got nil")
	}

	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("Expected AppError, got %T", err)
	}

	if appErr.Code != "NOT_FREE_COURSE" {
		t.Errorf("Expected error code NOT_FREE_COURSE, got %v", appErr.Code)
	}
}

func TestRevokeEnrollment_Success(t *testing.T) {
	enrollmentRepo := newMockEnrollmentRepo()
	courseRepo := newMockCourseRepo()
	lessonRepo := newMockLessonRepo()
	videoRepo := newMockVideoRepo()
	userRepo := newMockUserRepo()
	signingKeyStore := &mockSigningKeyStore{}
	storageClient := newMockStorageClient()

	lessonProgressRepo := newMockLessonProgressRepo()

	service := NewService(enrollmentRepo, lessonProgressRepo, courseRepo, lessonRepo, videoRepo, userRepo, signingKeyStore, storageClient, "videos")

	// Setup test data
	studentID := uuid.New()
	courseID := uuid.New()

	enrollment := &enrollments.Enrollment{
		ID:        uuid.New(),
		StudentID: studentID,
		CourseID:  courseID,
		Status:    enrollments.EnrollmentStatusActive,
	}
	key := studentID.String() + ":" + courseID.String()
	enrollmentRepo.enrollments[key] = enrollment

	// Execute
	cmd := RevokeEnrollmentCommand{
		StudentID: studentID,
		CourseID:  courseID,
		Status:    "cancelled",
	}

	err := service.RevokeEnrollment(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify enrollment status was updated
	updated, _ := enrollmentRepo.FindByStudentAndCourse(context.Background(), studentID, courseID)
	if updated.Status != enrollments.EnrollmentStatusCancelled {
		t.Errorf("Expected status cancelled, got %v", updated.Status)
	}
}

// Test GetStreamingSignedURL use case

func TestGetStreamingSignedURL_FreePreview_Success(t *testing.T) {
	enrollmentRepo := newMockEnrollmentRepo()
	courseRepo := newMockCourseRepo()
	lessonRepo := newMockLessonRepo()
	videoRepo := newMockVideoRepo()
	userRepo := newMockUserRepo()
	signingKeyStore := &mockSigningKeyStore{}
	storageClient := newMockStorageClient()

	lessonProgressRepo := newMockLessonProgressRepo()

	service := NewService(enrollmentRepo, lessonProgressRepo, courseRepo, lessonRepo, videoRepo, userRepo, signingKeyStore, storageClient, "videos")

	// Setup test data
	userID := uuid.New()
	lessonID := uuid.New()
	videoID := uuid.New()
	courseID := uuid.New()

	user := &auth.User{
		ID:              userID,
		Role:            "student",
		ProfileComplete: false, // Profile not complete, but should work for free preview
	}
	userRepo.users[userID] = user

	lesson := &courses.Lesson{
		ID:            lessonID,
		VideoID:       &videoID,
		IsFreePreview: true, // Free preview lesson
	}
	lessonRepo.lessons[lessonID] = lesson

	video := &courses.Video{
		ID:        videoID,
		CourseID:  courseID,
		RustFSKey: "videos/test-video.mp4",
		Status:    courses.VideoStatusReady,
	}
	videoRepo.videos[videoID] = video

	// Execute
	cmd := GetStreamingSignedURLCommand{
		UserID:   userID,
		LessonID: lessonID,
	}

	result, err := service.GetStreamingSignedURL(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.SignedURL == "" {
		t.Error("Expected signed URL, got empty string")
	}

	if result.ExpiresAt.IsZero() {
		t.Error("Expected expiry time, got zero time")
	}
}

func TestGetStreamingSignedURL_EnrolledStudent_Success(t *testing.T) {
	enrollmentRepo := newMockEnrollmentRepo()
	courseRepo := newMockCourseRepo()
	lessonRepo := newMockLessonRepo()
	videoRepo := newMockVideoRepo()
	userRepo := newMockUserRepo()
	signingKeyStore := &mockSigningKeyStore{}
	storageClient := newMockStorageClient()

	lessonProgressRepo := newMockLessonProgressRepo()

	service := NewService(enrollmentRepo, lessonProgressRepo, courseRepo, lessonRepo, videoRepo, userRepo, signingKeyStore, storageClient, "videos")

	// Setup test data
	userID := uuid.New()
	lessonID := uuid.New()
	videoID := uuid.New()
	courseID := uuid.New()

	user := &auth.User{
		ID:              userID,
		Role:            "student",
		ProfileComplete: true, // Profile complete
	}
	userRepo.users[userID] = user

	lesson := &courses.Lesson{
		ID:            lessonID,
		VideoID:       &videoID,
		IsFreePreview: false, // Not a free preview
	}
	lessonRepo.lessons[lessonID] = lesson

	video := &courses.Video{
		ID:        videoID,
		CourseID:  courseID,
		RustFSKey: "videos/test-video.mp4",
		Status:    courses.VideoStatusReady,
	}
	videoRepo.videos[videoID] = video

	enrollment := &enrollments.Enrollment{
		ID:        uuid.New(),
		StudentID: userID,
		CourseID:  courseID,
		Status:    enrollments.EnrollmentStatusActive,
	}
	key := userID.String() + ":" + courseID.String()
	enrollmentRepo.enrollments[key] = enrollment

	// Execute
	cmd := GetStreamingSignedURLCommand{
		UserID:   userID,
		LessonID: lessonID,
	}

	result, err := service.GetStreamingSignedURL(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.SignedURL == "" {
		t.Error("Expected signed URL, got empty string")
	}
}

func TestGetStreamingSignedURL_ProfileIncomplete_NonPreview(t *testing.T) {
	enrollmentRepo := newMockEnrollmentRepo()
	courseRepo := newMockCourseRepo()
	lessonRepo := newMockLessonRepo()
	videoRepo := newMockVideoRepo()
	userRepo := newMockUserRepo()
	signingKeyStore := &mockSigningKeyStore{}
	storageClient := newMockStorageClient()

	lessonProgressRepo := newMockLessonProgressRepo()

	service := NewService(enrollmentRepo, lessonProgressRepo, courseRepo, lessonRepo, videoRepo, userRepo, signingKeyStore, storageClient, "videos")

	// Setup test data
	userID := uuid.New()
	lessonID := uuid.New()
	videoID := uuid.New()
	courseID := uuid.New()

	user := &auth.User{
		ID:              userID,
		Role:            "student",
		ProfileComplete: false, // Profile not complete
	}
	userRepo.users[userID] = user

	lesson := &courses.Lesson{
		ID:            lessonID,
		VideoID:       &videoID,
		IsFreePreview: false, // Not a free preview - requires profile complete
	}
	lessonRepo.lessons[lessonID] = lesson

	video := &courses.Video{
		ID:        videoID,
		CourseID:  courseID,
		RustFSKey: "videos/test-video.mp4",
		Status:    courses.VideoStatusReady,
	}
	videoRepo.videos[videoID] = video

	// Execute
	cmd := GetStreamingSignedURLCommand{
		UserID:   userID,
		LessonID: lessonID,
	}

	_, err := service.GetStreamingSignedURL(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("Expected error for incomplete profile, got nil")
	}

	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("Expected AppError, got %T", err)
	}

	if appErr.Code != "PROFILE_INCOMPLETE" {
		t.Errorf("Expected error code PROFILE_INCOMPLETE, got %v", appErr.Code)
	}
}

func TestGetStreamingSignedURL_NotEnrolled(t *testing.T) {
	enrollmentRepo := newMockEnrollmentRepo()
	courseRepo := newMockCourseRepo()
	lessonRepo := newMockLessonRepo()
	videoRepo := newMockVideoRepo()
	userRepo := newMockUserRepo()
	signingKeyStore := &mockSigningKeyStore{}
	storageClient := newMockStorageClient()

	lessonProgressRepo := newMockLessonProgressRepo()

	service := NewService(enrollmentRepo, lessonProgressRepo, courseRepo, lessonRepo, videoRepo, userRepo, signingKeyStore, storageClient, "videos")

	// Setup test data
	userID := uuid.New()
	lessonID := uuid.New()
	videoID := uuid.New()
	courseID := uuid.New()

	user := &auth.User{
		ID:              userID,
		Role:            "student",
		ProfileComplete: true,
	}
	userRepo.users[userID] = user

	lesson := &courses.Lesson{
		ID:            lessonID,
		VideoID:       &videoID,
		IsFreePreview: false, // Not a free preview
	}
	lessonRepo.lessons[lessonID] = lesson

	video := &courses.Video{
		ID:        videoID,
		CourseID:  courseID,
		RustFSKey: "videos/test-video.mp4",
		Status:    courses.VideoStatusReady,
	}
	videoRepo.videos[videoID] = video

	// No enrollment created

	// Execute
	cmd := GetStreamingSignedURLCommand{
		UserID:   userID,
		LessonID: lessonID,
	}

	_, err := service.GetStreamingSignedURL(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("Expected error for not enrolled, got nil")
	}

	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("Expected AppError, got %T", err)
	}

	if appErr.Code != "NOT_ENROLLED" {
		t.Errorf("Expected error code NOT_ENROLLED, got %v", appErr.Code)
	}
}

func TestGetStreamingSignedURL_EnrollmentRevoked(t *testing.T) {
	enrollmentRepo := newMockEnrollmentRepo()
	courseRepo := newMockCourseRepo()
	lessonRepo := newMockLessonRepo()
	videoRepo := newMockVideoRepo()
	userRepo := newMockUserRepo()
	signingKeyStore := &mockSigningKeyStore{}
	storageClient := newMockStorageClient()

	lessonProgressRepo := newMockLessonProgressRepo()

	service := NewService(enrollmentRepo, lessonProgressRepo, courseRepo, lessonRepo, videoRepo, userRepo, signingKeyStore, storageClient, "videos")

	// Setup test data
	userID := uuid.New()
	lessonID := uuid.New()
	videoID := uuid.New()
	courseID := uuid.New()

	user := &auth.User{
		ID:              userID,
		Role:            "student",
		ProfileComplete: true,
	}
	userRepo.users[userID] = user

	lesson := &courses.Lesson{
		ID:            lessonID,
		VideoID:       &videoID,
		IsFreePreview: false,
	}
	lessonRepo.lessons[lessonID] = lesson

	video := &courses.Video{
		ID:        videoID,
		CourseID:  courseID,
		RustFSKey: "videos/test-video.mp4",
		Status:    courses.VideoStatusReady,
	}
	videoRepo.videos[videoID] = video

	enrollment := &enrollments.Enrollment{
		ID:        uuid.New(),
		StudentID: userID,
		CourseID:  courseID,
		Status:    enrollments.EnrollmentStatusCancelled, // Cancelled enrollment
	}
	key := userID.String() + ":" + courseID.String()
	enrollmentRepo.enrollments[key] = enrollment

	// Execute
	cmd := GetStreamingSignedURLCommand{
		UserID:   userID,
		LessonID: lessonID,
	}

	_, err := service.GetStreamingSignedURL(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("Expected error for revoked enrollment, got nil")
	}

	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("Expected AppError, got %T", err)
	}

	if appErr.Code != "ENROLLMENT_REVOKED" {
		t.Errorf("Expected error code ENROLLMENT_REVOKED, got %v", appErr.Code)
	}
}

func TestGetStreamingSignedURL_VideoNotReady(t *testing.T) {
	enrollmentRepo := newMockEnrollmentRepo()
	courseRepo := newMockCourseRepo()
	lessonRepo := newMockLessonRepo()
	videoRepo := newMockVideoRepo()
	userRepo := newMockUserRepo()
	signingKeyStore := &mockSigningKeyStore{}
	storageClient := newMockStorageClient()

	lessonProgressRepo := newMockLessonProgressRepo()

	service := NewService(enrollmentRepo, lessonProgressRepo, courseRepo, lessonRepo, videoRepo, userRepo, signingKeyStore, storageClient, "videos")

	// Setup test data
	userID := uuid.New()
	lessonID := uuid.New()
	videoID := uuid.New()
	courseID := uuid.New()

	user := &auth.User{
		ID:              userID,
		Role:            "student",
		ProfileComplete: true,
	}
	userRepo.users[userID] = user

	lesson := &courses.Lesson{
		ID:            lessonID,
		VideoID:       &videoID,
		IsFreePreview: true,
	}
	lessonRepo.lessons[lessonID] = lesson

	video := &courses.Video{
		ID:        videoID,
		CourseID:  courseID,
		RustFSKey: "videos/test-video.mp4",
		Status:    courses.VideoStatusProcessing, // Video still processing
	}
	videoRepo.videos[videoID] = video

	// Execute
	cmd := GetStreamingSignedURLCommand{
		UserID:   userID,
		LessonID: lessonID,
	}

	_, err := service.GetStreamingSignedURL(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("Expected error for video not ready, got nil")
	}

	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("Expected AppError, got %T", err)
	}

	if appErr.Code != "VIDEO_NOT_READY" {
		t.Errorf("Expected error code VIDEO_NOT_READY, got %v", appErr.Code)
	}
}

func TestGetStreamingSignedURL_LessonNotFound(t *testing.T) {
	enrollmentRepo := newMockEnrollmentRepo()
	courseRepo := newMockCourseRepo()
	lessonRepo := newMockLessonRepo()
	videoRepo := newMockVideoRepo()
	userRepo := newMockUserRepo()
	signingKeyStore := &mockSigningKeyStore{}
	storageClient := newMockStorageClient()

	lessonProgressRepo := newMockLessonProgressRepo()

	service := NewService(enrollmentRepo, lessonProgressRepo, courseRepo, lessonRepo, videoRepo, userRepo, signingKeyStore, storageClient, "videos")

	// Setup test data
	userID := uuid.New()
	lessonID := uuid.New() // Lesson doesn't exist

	user := &auth.User{
		ID:              userID,
		Role:            "student",
		ProfileComplete: true,
	}
	userRepo.users[userID] = user

	// Execute
	cmd := GetStreamingSignedURLCommand{
		UserID:   userID,
		LessonID: lessonID,
	}

	_, err := service.GetStreamingSignedURL(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("Expected error for lesson not found, got nil")
	}

	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("Expected AppError, got %T", err)
	}

	if appErr.Code != "LESSON_NOT_FOUND" {
		t.Errorf("Expected error code LESSON_NOT_FOUND, got %v", appErr.Code)
	}
}

func TestGetStreamingSignedURL_NoVideo(t *testing.T) {
	enrollmentRepo := newMockEnrollmentRepo()
	courseRepo := newMockCourseRepo()
	lessonRepo := newMockLessonRepo()
	videoRepo := newMockVideoRepo()
	userRepo := newMockUserRepo()
	signingKeyStore := &mockSigningKeyStore{}
	storageClient := newMockStorageClient()

	lessonProgressRepo := newMockLessonProgressRepo()

	service := NewService(enrollmentRepo, lessonProgressRepo, courseRepo, lessonRepo, videoRepo, userRepo, signingKeyStore, storageClient, "videos")

	// Setup test data
	userID := uuid.New()
	lessonID := uuid.New()

	user := &auth.User{
		ID:              userID,
		Role:            "student",
		ProfileComplete: true,
	}
	userRepo.users[userID] = user

	lesson := &courses.Lesson{
		ID:            lessonID,
		VideoID:       nil, // No video attached
		IsFreePreview: true,
	}
	lessonRepo.lessons[lessonID] = lesson

	// Execute
	cmd := GetStreamingSignedURLCommand{
		UserID:   userID,
		LessonID: lessonID,
	}

	_, err := service.GetStreamingSignedURL(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("Expected error for no video, got nil")
	}

	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("Expected AppError, got %T", err)
	}

	if appErr.Code != "NO_VIDEO" {
		t.Errorf("Expected error code NO_VIDEO, got %v", appErr.Code)
	}
}

// Test UpdateLessonProgress use case

func TestUpdateLessonProgress_Success(t *testing.T) {
	enrollmentRepo := newMockEnrollmentRepo()
	lessonProgressRepo := newMockLessonProgressRepo()
	courseRepo := newMockCourseRepo()
	lessonRepo := newMockLessonRepo()
	videoRepo := newMockVideoRepo()
	userRepo := newMockUserRepo()
	signingKeyStore := &mockSigningKeyStore{}
	storageClient := newMockStorageClient()

	service := NewService(enrollmentRepo, lessonProgressRepo, courseRepo, lessonRepo, videoRepo, userRepo, signingKeyStore, storageClient, "videos")

	// Setup test data
	enrollmentID := uuid.New()
	lessonID := uuid.New()
	studentID := uuid.New()
	courseID := uuid.New()

	enrollment := &enrollments.Enrollment{
		ID:        enrollmentID,
		StudentID: studentID,
		CourseID:  courseID,
		Status:    enrollments.EnrollmentStatusActive,
	}
	key := studentID.String() + ":" + courseID.String()
	enrollmentRepo.enrollments[key] = enrollment

	// Execute
	cmd := UpdateLessonProgressCommand{
		EnrollmentID:    enrollmentID,
		LessonID:        lessonID,
		PositionSeconds: 120,
		WatchedPercent:  85.0,
		Completed:       true,
	}

	result, err := service.UpdateLessonProgress(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.EnrollmentID != enrollmentID {
		t.Errorf("Expected enrollment ID %v, got %v", enrollmentID, result.EnrollmentID)
	}

	if result.LessonID != lessonID {
		t.Errorf("Expected lesson ID %v, got %v", lessonID, result.LessonID)
	}

	if result.PositionSeconds != 120 {
		t.Errorf("Expected position 120, got %v", result.PositionSeconds)
	}

	if result.WatchedPercent != 85.0 {
		t.Errorf("Expected watched percent 85.0, got %v", result.WatchedPercent)
	}

	if !result.Completed {
		t.Error("Expected completed to be true")
	}

	if result.CompletedAt == nil {
		t.Error("Expected completed_at to be set")
	}

	// Verify progress was stored in repository
	progressKey := enrollmentID.String() + ":" + lessonID.String()
	storedProgress, exists := lessonProgressRepo.progress[progressKey]
	if !exists {
		t.Fatal("Expected progress to be stored in repository")
	}

	if !storedProgress.Completed {
		t.Error("Expected stored progress to be completed")
	}
}

func TestUpdateLessonProgress_NotCompleted_BelowThreshold(t *testing.T) {
	enrollmentRepo := newMockEnrollmentRepo()
	lessonProgressRepo := newMockLessonProgressRepo()
	courseRepo := newMockCourseRepo()
	lessonRepo := newMockLessonRepo()
	videoRepo := newMockVideoRepo()
	userRepo := newMockUserRepo()
	signingKeyStore := &mockSigningKeyStore{}
	storageClient := newMockStorageClient()

	service := NewService(enrollmentRepo, lessonProgressRepo, courseRepo, lessonRepo, videoRepo, userRepo, signingKeyStore, storageClient, "videos")

	// Setup test data
	enrollmentID := uuid.New()
	lessonID := uuid.New()
	studentID := uuid.New()
	courseID := uuid.New()

	enrollment := &enrollments.Enrollment{
		ID:        enrollmentID,
		StudentID: studentID,
		CourseID:  courseID,
		Status:    enrollments.EnrollmentStatusActive,
	}
	key := studentID.String() + ":" + courseID.String()
	enrollmentRepo.enrollments[key] = enrollment

	// Execute - watched_percent below 80% threshold
	cmd := UpdateLessonProgressCommand{
		EnrollmentID:    enrollmentID,
		LessonID:        lessonID,
		PositionSeconds: 60,
		WatchedPercent:  75.0, // Below 80% threshold
		Completed:       true,
	}

	result, err := service.UpdateLessonProgress(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Should NOT be marked as completed because watched_percent < 80
	if result.Completed {
		t.Error("Expected completed to be false when watched_percent < 80")
	}

	if result.CompletedAt != nil {
		t.Error("Expected completed_at to be nil when not completed")
	}
}

func TestUpdateLessonProgress_EnrollmentRevoked(t *testing.T) {
	enrollmentRepo := newMockEnrollmentRepo()
	lessonProgressRepo := newMockLessonProgressRepo()
	courseRepo := newMockCourseRepo()
	lessonRepo := newMockLessonRepo()
	videoRepo := newMockVideoRepo()
	userRepo := newMockUserRepo()
	signingKeyStore := &mockSigningKeyStore{}
	storageClient := newMockStorageClient()

	service := NewService(enrollmentRepo, lessonProgressRepo, courseRepo, lessonRepo, videoRepo, userRepo, signingKeyStore, storageClient, "videos")

	// Setup test data
	enrollmentID := uuid.New()
	lessonID := uuid.New()
	studentID := uuid.New()
	courseID := uuid.New()

	enrollment := &enrollments.Enrollment{
		ID:        enrollmentID,
		StudentID: studentID,
		CourseID:  courseID,
		Status:    enrollments.EnrollmentStatusCancelled, // Cancelled enrollment
	}
	key := studentID.String() + ":" + courseID.String()
	enrollmentRepo.enrollments[key] = enrollment

	// Execute
	cmd := UpdateLessonProgressCommand{
		EnrollmentID:    enrollmentID,
		LessonID:        lessonID,
		PositionSeconds: 120,
		WatchedPercent:  85.0,
		Completed:       true,
	}

	_, err := service.UpdateLessonProgress(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("Expected error for revoked enrollment, got nil")
	}

	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("Expected AppError, got %T", err)
	}

	if appErr.Code != "ENROLLMENT_REVOKED" {
		t.Errorf("Expected error code ENROLLMENT_REVOKED, got %v", appErr.Code)
	}
}

func TestUpdateLessonProgress_EnrollmentNotFound(t *testing.T) {
	enrollmentRepo := newMockEnrollmentRepo()
	lessonProgressRepo := newMockLessonProgressRepo()
	courseRepo := newMockCourseRepo()
	lessonRepo := newMockLessonRepo()
	videoRepo := newMockVideoRepo()
	userRepo := newMockUserRepo()
	signingKeyStore := &mockSigningKeyStore{}
	storageClient := newMockStorageClient()

	service := NewService(enrollmentRepo, lessonProgressRepo, courseRepo, lessonRepo, videoRepo, userRepo, signingKeyStore, storageClient, "videos")

	// Setup test data - no enrollment created
	enrollmentID := uuid.New()
	lessonID := uuid.New()

	// Execute
	cmd := UpdateLessonProgressCommand{
		EnrollmentID:    enrollmentID,
		LessonID:        lessonID,
		PositionSeconds: 120,
		WatchedPercent:  85.0,
		Completed:       true,
	}

	_, err := service.UpdateLessonProgress(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("Expected error for enrollment not found, got nil")
	}

	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("Expected AppError, got %T", err)
	}

	if appErr.Code != "ENROLLMENT_NOT_FOUND" {
		t.Errorf("Expected error code ENROLLMENT_NOT_FOUND, got %v", appErr.Code)
	}
}

func TestGetLessonProgress_Success(t *testing.T) {
	enrollmentRepo := newMockEnrollmentRepo()
	lessonProgressRepo := newMockLessonProgressRepo()
	courseRepo := newMockCourseRepo()
	lessonRepo := newMockLessonRepo()
	videoRepo := newMockVideoRepo()
	userRepo := newMockUserRepo()
	signingKeyStore := &mockSigningKeyStore{}
	storageClient := newMockStorageClient()

	service := NewService(enrollmentRepo, lessonProgressRepo, courseRepo, lessonRepo, videoRepo, userRepo, signingKeyStore, storageClient, "videos")

	// Setup test data
	enrollmentID := uuid.New()
	lessonID := uuid.New()

	now := time.Now()
	progress := &enrollments.LessonProgress{
		ID:              uuid.New(),
		EnrollmentID:    enrollmentID,
		LessonID:        lessonID,
		PositionSeconds: 120,
		WatchedPercent:  85.0,
		Completed:       true,
		CompletedAt:     &now,
		LastWatchedAt:   &now,
	}
	key := enrollmentID.String() + ":" + lessonID.String()
	lessonProgressRepo.progress[key] = progress

	// Execute
	result, err := service.GetLessonProgress(context.Background(), enrollmentID, lessonID)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.EnrollmentID != enrollmentID {
		t.Errorf("Expected enrollment ID %v, got %v", enrollmentID, result.EnrollmentID)
	}

	if result.LessonID != lessonID {
		t.Errorf("Expected lesson ID %v, got %v", lessonID, result.LessonID)
	}

	if result.PositionSeconds != 120 {
		t.Errorf("Expected position 120, got %v", result.PositionSeconds)
	}

	if !result.Completed {
		t.Error("Expected completed to be true")
	}
}

func TestGetLessonProgress_NotFound(t *testing.T) {
	enrollmentRepo := newMockEnrollmentRepo()
	lessonProgressRepo := newMockLessonProgressRepo()
	courseRepo := newMockCourseRepo()
	lessonRepo := newMockLessonRepo()
	videoRepo := newMockVideoRepo()
	userRepo := newMockUserRepo()
	signingKeyStore := &mockSigningKeyStore{}
	storageClient := newMockStorageClient()

	service := NewService(enrollmentRepo, lessonProgressRepo, courseRepo, lessonRepo, videoRepo, userRepo, signingKeyStore, storageClient, "videos")

	// Setup test data - no progress created
	enrollmentID := uuid.New()
	lessonID := uuid.New()

	// Execute
	_, err := service.GetLessonProgress(context.Background(), enrollmentID, lessonID)

	// Assert
	if err == nil {
		t.Fatal("Expected error for progress not found, got nil")
	}

	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("Expected AppError, got %T", err)
	}

	if appErr.Code != "LESSON_PROGRESS_NOT_FOUND" {
		t.Errorf("Expected error code LESSON_PROGRESS_NOT_FOUND, got %v", appErr.Code)
	}
}
