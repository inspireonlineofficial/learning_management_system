package enrollments

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"strings"
	"testing"
	"time"

	"lms-backend/internal/domain/auth"
	"lms-backend/internal/domain/courses"
	"lms-backend/internal/domain/enrollments"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// Regression tests for issues found in the November 2026 audit.

type progressFixture struct {
	svc          *service
	enrollRepo   *mockEnrollmentRepo
	progressRepo *mockLessonProgressRepo
	lessonRepo   *mockLessonRepo
	videoRepo    *mockVideoRepo
	userRepo     *mockUserRepo
	courseRepo   *mockCourseRepo
	storage      *mockStorageClient
	studentID    uuid.UUID
	courseID     uuid.UUID
	enrollmentID uuid.UUID
}

func newProgressFixture(t *testing.T) *progressFixture {
	t.Helper()
	f := &progressFixture{
		enrollRepo:   newMockEnrollmentRepo(),
		progressRepo: newMockLessonProgressRepo(),
		lessonRepo:   newMockLessonRepo(),
		videoRepo:    newMockVideoRepo(),
		userRepo:     newMockUserRepo(),
		courseRepo:   newMockCourseRepo(),
		storage:      newMockStorageClient(),
		studentID:    uuid.New(),
		courseID:     uuid.New(),
		enrollmentID: uuid.New(),
	}
	f.svc = NewService(f.enrollRepo, f.progressRepo, f.courseRepo, f.lessonRepo, f.videoRepo, f.userRepo, &mockSigningKeyStore{}, f.storage, "videos").(*service)
	f.enrollRepo.enrollments[f.studentID.String()+":"+f.courseID.String()] = &enrollments.Enrollment{
		ID: f.enrollmentID, StudentID: f.studentID, CourseID: f.courseID, Status: enrollments.EnrollmentStatusActive,
	}
	f.userRepo.users[f.studentID] = &auth.User{ID: f.studentID, Role: "student", ProfileComplete: true}
	return f
}

func (f *progressFixture) addVideoLesson(courseID uuid.UUID, status courses.LessonStatus, videoDuration int) *courses.Lesson {
	videoID := uuid.New()
	f.videoRepo.videos[videoID] = &courses.Video{ID: videoID, CourseID: courseID, Status: courses.VideoStatusReady, DurationSeconds: videoDuration, RustFSKey: "videos/" + videoID.String() + ".mp4"}
	lesson := &courses.Lesson{ID: uuid.New(), VideoID: &videoID, Type: courses.LessonTypeVideo, Status: status}
	f.lessonRepo.lessons[lesson.ID] = lesson
	return lesson
}

func appCode(err error) string {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return ""
}

func TestAudit_ProgressRejectsLessonFromAnotherCourse(t *testing.T) {
	f := newProgressFixture(t)
	foreign := f.addVideoLesson(uuid.New(), courses.LessonStatusPublished, 0)

	_, err := f.svc.UpdateLessonProgress(context.Background(), UpdateLessonProgressCommand{
		EnrollmentID: f.enrollmentID, LessonID: foreign.ID, WatchedPercent: 100, Completed: true,
	})
	if appCode(err) != "LESSON_NOT_FOUND" {
		t.Fatalf("expected LESSON_NOT_FOUND for a lesson of another course, got %v", err)
	}
	if len(f.progressRepo.progress) != 0 {
		t.Fatalf("no progress must be stored for foreign lessons")
	}
}

func TestAudit_ProgressRejectsDraftLesson(t *testing.T) {
	f := newProgressFixture(t)
	draft := f.addVideoLesson(f.courseID, courses.LessonStatusDraft, 0)
	_, err := f.svc.UpdateLessonProgress(context.Background(), UpdateLessonProgressCommand{
		EnrollmentID: f.enrollmentID, LessonID: draft.ID, WatchedPercent: 50,
	})
	if appCode(err) != "LESSON_NOT_FOUND" {
		t.Fatalf("expected LESSON_NOT_FOUND for draft lesson, got %v", err)
	}
}

func TestAudit_ProgressValidatesInput(t *testing.T) {
	f := newProgressFixture(t)
	lesson := f.addVideoLesson(f.courseID, courses.LessonStatusPublished, 0)
	for _, tc := range []struct {
		name string
		cmd  UpdateLessonProgressCommand
		code string
	}{
		{"percent too high", UpdateLessonProgressCommand{WatchedPercent: 5000}, "INVALID_WATCHED_PERCENT"},
		{"negative percent", UpdateLessonProgressCommand{WatchedPercent: -1}, "INVALID_WATCHED_PERCENT"},
		{"negative position", UpdateLessonProgressCommand{PositionSeconds: -5}, "INVALID_POSITION"},
	} {
		tc.cmd.EnrollmentID = f.enrollmentID
		tc.cmd.LessonID = lesson.ID
		if _, err := f.svc.UpdateLessonProgress(context.Background(), tc.cmd); appCode(err) != tc.code {
			t.Errorf("%s: expected %s, got %v", tc.name, tc.code, err)
		}
	}
}

func TestAudit_VideoCompletionRequiresPlausibleWatchTime(t *testing.T) {
	f := newProgressFixture(t)
	lesson := f.addVideoLesson(f.courseID, courses.LessonStatusPublished, 600) // 10 minute video

	// Opening the lesson and immediately claiming completion is not accepted.
	res, err := f.svc.UpdateLessonProgress(context.Background(), UpdateLessonProgressCommand{
		EnrollmentID: f.enrollmentID, LessonID: lesson.ID, WatchedPercent: 100, Completed: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Completed {
		t.Fatalf("completion must be deferred when the lesson was opened seconds ago")
	}

	// After a plausible amount of time (>= 25% of the duration) it is accepted.
	key := f.enrollmentID.String() + ":" + lesson.ID.String()
	started := time.Now().Add(-3 * time.Minute)
	f.progressRepo.progress[key].StartedAt = &started
	res, err = f.svc.UpdateLessonProgress(context.Background(), UpdateLessonProgressCommand{
		EnrollmentID: f.enrollmentID, LessonID: lesson.ID, WatchedPercent: 95, Completed: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Completed {
		t.Fatalf("completion should be accepted after plausible watch time")
	}
}

func TestAudit_StreamingRejectsBorrowedVideo(t *testing.T) {
	f := newProgressFixture(t)
	// The lesson's chapter belongs to course A (resolved via placement), but
	// its video belongs to course B.
	courseB := uuid.New()
	videoID := uuid.New()
	f.videoRepo.videos[videoID] = &courses.Video{ID: videoID, CourseID: courseB, Status: courses.VideoStatusReady, RustFSKey: "videos/x.mp4"}
	lesson := &courses.Lesson{ID: uuid.New(), VideoID: &videoID, IsFreePreview: true, Status: courses.LessonStatusPublished}
	repo := &placementLessonRepo{mockLessonRepo: f.lessonRepo, placements: map[uuid.UUID]*courses.LessonPlacement{
		lesson.ID: {CourseID: f.courseID, CourseStatus: courses.CourseStatusPublished, ModulePublished: true},
	}}
	f.lessonRepo.lessons[lesson.ID] = lesson
	f.svc.lessonRepo = repo

	_, err := f.svc.GetStreamingSignedURL(context.Background(), GetStreamingSignedURLCommand{UserID: f.studentID, LessonID: lesson.ID})
	if appCode(err) != "ACCESS_DENIED" {
		t.Fatalf("expected ACCESS_DENIED when a lesson references another course's video, got %v", err)
	}
}

func TestAudit_StreamingFreeLessonRequiresPublishedCourse(t *testing.T) {
	f := newProgressFixture(t)
	lesson := f.addVideoLesson(f.courseID, courses.LessonStatusPublished, 0)
	lesson.IsFree = true
	repo := &placementLessonRepo{mockLessonRepo: f.lessonRepo, placements: map[uuid.UUID]*courses.LessonPlacement{
		lesson.ID: {CourseID: f.courseID, CourseStatus: courses.CourseStatusDraft, ModulePublished: true},
	}}
	f.svc.lessonRepo = repo
	_, err := f.svc.GetStreamingSignedURL(context.Background(), GetStreamingSignedURLCommand{UserID: uuid.New(), LessonID: lesson.ID})
	if appCode(err) != "USER_NOT_FOUND" && appCode(err) != "LESSON_NOT_FOUND" {
		t.Fatalf("expected the free lesson of a draft course to be unavailable, got %v", err)
	}
	other := uuid.New()
	f.userRepo.users[other] = &auth.User{ID: other, Role: "student"}
	_, err = f.svc.GetStreamingSignedURL(context.Background(), GetStreamingSignedURLCommand{UserID: other, LessonID: lesson.ID})
	if appCode(err) != "LESSON_NOT_FOUND" {
		t.Fatalf("expected LESSON_NOT_FOUND, got %v", err)
	}
}

func TestAudit_HLSPlaylistSigning(t *testing.T) {
	f := newProgressFixture(t)
	f.svc.SetStreamSigningKey([]byte("test-key"))
	lesson := f.addVideoLesson(f.courseID, courses.LessonStatusPublished, 0)
	video := f.videoRepo.videos[*lesson.VideoID]
	video.HLSManifestKey = "videos/videos/abc.mp4/hls/master.m3u8"

	storage := &hlsStorage{mockStorageClient: f.storage, objects: map[string]string{
		"videos/videos/abc.mp4/hls/master.m3u8":      "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=896000\nv0/playlist.m3u8\n",
		"videos/videos/abc.mp4/hls/v0/playlist.m3u8": "#EXTM3U\n#EXTINF:4.0,\nseg_000.ts\n#EXTINF:4.0,\n../../../secret.ts\n#EXT-X-ENDLIST\n",
	}}
	f.svc.storageClient = storage

	resp, err := f.svc.GetStreamingSignedURL(context.Background(), GetStreamingSignedURLCommand{UserID: f.studentID, LessonID: lesson.ID})
	if err != nil {
		t.Fatalf("signed url: %v", err)
	}
	if !resp.HasHLS || !strings.HasPrefix(resp.HLSURL, "/v1/stream/hls/"+video.ID.String()+"/master.m3u8?") {
		t.Fatalf("expected a backend HLS URL, got %q", resp.HLSURL)
	}
	u, _ := url.Parse(resp.HLSURL)
	exp, sig := u.Query().Get("exp"), u.Query().Get("sig")

	master, err := f.svc.GetHLSPlaylist(context.Background(), video.ID, "", exp, sig)
	if err != nil {
		t.Fatalf("master playlist: %v", err)
	}
	if !strings.Contains(string(master), "v0/playlist.m3u8?exp=") {
		t.Fatalf("master playlist should carry the signature to renditions:\n%s", master)
	}

	variant, err := f.svc.GetHLSPlaylist(context.Background(), video.ID, "v0", exp, sig)
	if err != nil {
		t.Fatalf("variant playlist: %v", err)
	}
	if !strings.Contains(string(variant), "https://example.com/presigned/videos/videos/abc.mp4/hls/v0/seg_000.ts") {
		t.Fatalf("segments should be presigned:\n%s", variant)
	}
	if strings.Contains(string(variant), "secret") {
		t.Fatalf("path traversal segment must be dropped:\n%s", variant)
	}

	if _, err := f.svc.GetHLSPlaylist(context.Background(), video.ID, "", exp, sig+"00"); appCode(err) != "STREAM_URL_INVALID" {
		t.Fatalf("tampered signature must be rejected, got %v", err)
	}
	if _, err := f.svc.GetHLSPlaylist(context.Background(), uuid.New(), "", exp, sig); appCode(err) != "STREAM_URL_INVALID" {
		t.Fatalf("signature must be bound to the video, got %v", err)
	}
	if _, err := f.svc.GetHLSPlaylist(context.Background(), video.ID, "../x", exp, sig); appCode(err) != "NOT_FOUND" {
		t.Fatalf("invalid rendition must be rejected, got %v", err)
	}
	if _, err := f.svc.GetHLSPlaylist(context.Background(), video.ID, "", "1", f.svc.signHLS(video.ID, 1)); appCode(err) != "STREAM_URL_EXPIRED" {
		t.Fatalf("expired URL must be rejected, got %v", err)
	}
}

type placementLessonRepo struct {
	*mockLessonRepo
	placements map[uuid.UUID]*courses.LessonPlacement
}

func (r *placementLessonRepo) FindLessonPlacement(ctx context.Context, lessonID uuid.UUID) (*courses.LessonPlacement, error) {
	if p, ok := r.placements[lessonID]; ok {
		return p, nil
	}
	return nil, errors.New("not found")
}

type hlsStorage struct {
	*mockStorageClient
	objects map[string]string
}

func (h *hlsStorage) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	body, ok := h.objects[key]
	if !ok {
		return nil, errors.New("missing")
	}
	return io.NopCloser(bytes.NewReader([]byte(body))), nil
}

func TestAudit_BunnyEdgeDeliveryURL(t *testing.T) {
	f := newProgressFixture(t)
	f.svc.SetEdgeDelivery(EdgeDeliveryConfig{Mode: "bunny", Host: "https://video.example.com/", TokenKey: "secret"})
	lesson := f.addVideoLesson(f.courseID, courses.LessonStatusPublished, 0)
	f.videoRepo.videos[*lesson.VideoID].HLSManifestKey = "videos/videos/abc.mp4/hls/1700000000/master.m3u8"

	resp, err := f.svc.GetStreamingSignedURL(context.Background(), GetStreamingSignedURLCommand{UserID: f.studentID, LessonID: lesson.ID})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Delivery != "edge" || !strings.HasPrefix(resp.HLSURL, "https://video.example.com/bcdn_token=HS256-") {
		t.Fatalf("unexpected edge URL %q (%s)", resp.HLSURL, resp.Delivery)
	}
	if !strings.Contains(resp.HLSURL, "token_path=%2Fvideos%2Fvideos%2Fabc.mp4%2Fhls%2F1700000000%2F") ||
		!strings.HasSuffix(resp.HLSURL, "/videos/videos/abc.mp4/hls/1700000000/master.m3u8") {
		t.Fatalf("token must be scoped to the manifest directory: %q", resp.HLSURL)
	}
	// Deterministic for identical inputs; changes with expiry.
	a := bunnySignDirectory("h", "k", "a/b/c.m3u8", 100)
	if a != bunnySignDirectory("h", "k", "a/b/c.m3u8", 100) || a == bunnySignDirectory("h", "k", "a/b/c.m3u8", 101) {
		t.Fatalf("token signing is not deterministic/expiry-bound")
	}
}
