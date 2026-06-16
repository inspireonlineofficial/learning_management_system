package points

import (
	"context"
	"testing"
	"time"

	domainpoints "lms-backend/internal/domain/points"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// ─── Mock implementations ────────────────────────────────────────────────────

// mockPointEventRepo implements domainpoints.PointEventRepository
type mockPointEventRepo struct {
	events []*domainpoints.PointEvent
}

func newMockPointEventRepo() *mockPointEventRepo {
	return &mockPointEventRepo{events: make([]*domainpoints.PointEvent, 0)}
}

func (m *mockPointEventRepo) Create(ctx context.Context, event *domainpoints.PointEvent) error {
	m.events = append(m.events, event)
	return nil
}

func (m *mockPointEventRepo) FindByID(ctx context.Context, id uuid.UUID) (*domainpoints.PointEvent, error) {
	for _, e := range m.events {
		if e.ID == id {
			return e, nil
		}
	}
	return nil, nil
}

func (m *mockPointEventRepo) FindByStudentID(ctx context.Context, studentID uuid.UUID, page, limit int) ([]*domainpoints.PointEvent, int, error) {
	var result []*domainpoints.PointEvent
	for _, e := range m.events {
		if e.StudentID == studentID {
			result = append(result, e)
		}
	}
	return result, len(result), nil
}

func (m *mockPointEventRepo) ExistsForSourceOnDay(ctx context.Context, studentID, sourceID uuid.UUID, eventType domainpoints.PointEventType, day time.Time) (bool, error) {
	dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)
	for _, e := range m.events {
		if e.StudentID == studentID &&
			e.SourceID == sourceID &&
			e.Type == eventType &&
			!e.EarnedAt.Before(dayStart) &&
			e.EarnedAt.Before(dayEnd) {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockPointEventRepo) ExistsPassingForSource(ctx context.Context, studentID, sourceID uuid.UUID, eventType domainpoints.PointEventType) (bool, error) {
	for _, e := range m.events {
		if e.StudentID == studentID && e.SourceID == sourceID && e.Type == eventType {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockPointEventRepo) SumByStudentID(ctx context.Context, studentID uuid.UUID) (int, error) {
	total := 0
	for _, e := range m.events {
		if e.StudentID == studentID {
			total += e.Points + e.BonusPoints
		}
	}
	return total, nil
}

func (m *mockPointEventRepo) SumByStudentIDSince(ctx context.Context, studentID uuid.UUID, since time.Time) (int, error) {
	total := 0
	for _, e := range m.events {
		if e.StudentID == studentID && !e.EarnedAt.Before(since) {
			total += e.Points + e.BonusPoints
		}
	}
	return total, nil
}

// mockPointsConfigRepo implements domainpoints.PointsConfigRepository
type mockPointsConfigRepo struct {
	config *domainpoints.PointsConfig
}

func newMockPointsConfigRepo() *mockPointsConfigRepo {
	return &mockPointsConfigRepo{
		config: &domainpoints.PointsConfig{
			ID:                      1,
			PointsPerVideo:          10,
			PointsPerQuizPass:       20,
			BonusPointsPerfectScore: 10,
		},
	}
}

func (m *mockPointsConfigRepo) Get(ctx context.Context) (*domainpoints.PointsConfig, error) {
	return m.config, nil
}

func (m *mockPointsConfigRepo) Update(ctx context.Context, config *domainpoints.PointsConfig) error {
	m.config = config
	return nil
}

// mockPointsRankRepo implements PointsRankRepository
type mockPointsRankRepo struct{}

func (m *mockPointsRankRepo) CountStudentsWithMoreTotalPoints(ctx context.Context, totalPoints int) (int, error) {
	return 0, nil
}

func (m *mockPointsRankRepo) CountStudentsWithMoreWeeklyPoints(ctx context.Context, weeklyPoints int, weekStart time.Time) (int, error) {
	return 0, nil
}

func (m *mockPointsRankRepo) FindEventsForDay(ctx context.Context, studentID uuid.UUID, day time.Time) ([]*domainpoints.PointEvent, error) {
	return nil, nil
}

// mockLeaderboardStore implements LeaderboardStore
type mockLeaderboardStore struct {
	scores   map[string]map[uuid.UUID]float64 // key -> memberID -> score
	optedOut map[uuid.UUID]bool
}

func newMockLeaderboardStore() *mockLeaderboardStore {
	return &mockLeaderboardStore{
		scores:   make(map[string]map[uuid.UUID]float64),
		optedOut: make(map[uuid.UUID]bool),
	}
}

func (m *mockLeaderboardStore) AddScore(ctx context.Context, key string, memberID uuid.UUID, score float64) error {
	if m.scores[key] == nil {
		m.scores[key] = make(map[uuid.UUID]float64)
	}
	m.scores[key][memberID] += score
	return nil
}

func (m *mockLeaderboardStore) GetTopN(ctx context.Context, key string, n int) ([]LeaderboardRawEntry, error) {
	members := m.scores[key]
	entries := make([]LeaderboardRawEntry, 0, len(members))
	for id, score := range members {
		entries = append(entries, LeaderboardRawEntry{MemberID: id, Score: score})
	}
	// Sort descending by score (simple insertion sort for test purposes)
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j].Score > entries[j-1].Score; j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}
	if n < len(entries) {
		entries = entries[:n]
	}
	return entries, nil
}

func (m *mockLeaderboardStore) ResetWeekly(ctx context.Context) error {
	delete(m.scores, "leaderboard:weekly")
	return nil
}

func (m *mockLeaderboardStore) GetOptOutStatus(ctx context.Context, studentID uuid.UUID) (bool, error) {
	return m.optedOut[studentID], nil
}

func (m *mockLeaderboardStore) SetOptOutStatus(ctx context.Context, studentID uuid.UUID, optOut bool) error {
	m.optedOut[studentID] = optOut
	return nil
}

// mockStudentNameResolver implements StudentNameResolver
type mockStudentNameResolver struct {
	names map[uuid.UUID]string
}

func newMockStudentNameResolver() *mockStudentNameResolver {
	return &mockStudentNameResolver{names: make(map[uuid.UUID]string)}
}

func (m *mockStudentNameResolver) GetDisplayName(ctx context.Context, studentID uuid.UUID) (string, error) {
	if name, ok := m.names[studentID]; ok {
		return name, nil
	}
	return "Student " + studentID.String()[:8], nil
}

// mockAuditLogger implements AuditLogger
type mockAuditLogger struct{}

func (m *mockAuditLogger) LogAction(ctx context.Context, actorID uuid.UUID, actorName, action, targetType string, targetID uuid.UUID, metadata map[string]interface{}, ipAddress string) error {
	return nil
}

// ─── Helper: build a service with all mocks ──────────────────────────────────

type pointsPropDeps struct {
	eventRepo    *mockPointEventRepo
	configRepo   *mockPointsConfigRepo
	rankRepo     *mockPointsRankRepo
	leaderboard  *mockLeaderboardStore
	nameResolver *mockStudentNameResolver
}

func newPointsPropDeps() *pointsPropDeps {
	return &pointsPropDeps{
		eventRepo:    newMockPointEventRepo(),
		configRepo:   newMockPointsConfigRepo(),
		rankRepo:     &mockPointsRankRepo{},
		leaderboard:  newMockLeaderboardStore(),
		nameResolver: newMockStudentNameResolver(),
	}
}

func (d *pointsPropDeps) service() Service {
	return NewService(
		d.eventRepo,
		d.configRepo,
		d.rankRepo,
		d.leaderboard,
		d.nameResolver,
		&mockAuditLogger{},
	)
}

// ─── Property 48 ─────────────────────────────────────────────────────────────

// TestProperty48_VideoPointsAwardedExactlyOncePerLessonPerCalendarDay verifies
// that AwardVideoPoints awards points exactly once per student+lesson per UTC
// calendar day, and awards again on a different calendar day.
//
// **Validates: Requirements 17.1, 17.2**
func TestProperty48_VideoPointsAwardedExactlyOncePerLessonPerCalendarDay(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		deps := newPointsPropDeps()
		svc := deps.service()

		studentID := uuid.New()
		lessonID := uuid.New()

		// Generate 2–5 calls on the same day
		numCalls := rapid.IntRange(2, 5).Draw(t, "num_calls")

		// Simulate "today" by pre-seeding the first event with today's timestamp.
		// The service uses time.Now() internally, so we rely on the mock's
		// ExistsForSourceOnDay which checks the actual EarnedAt timestamps.
		// All calls happen within the same test run (same UTC day).

		ctx := context.Background()

		// First call: must award points
		firstResult, err := svc.AwardVideoPoints(ctx, AwardVideoPointsCommand{
			StudentID:   studentID,
			LessonID:    lessonID,
			SourceTitle: "Lesson 1",
		})
		if err != nil {
			t.Fatalf("first AwardVideoPoints failed: %v", err)
		}
		if !firstResult.Awarded {
			t.Fatal("first call must award points (Requirement 17.1)")
		}
		if firstResult.Points <= 0 {
			t.Fatalf("first call must award positive points, got %d", firstResult.Points)
		}

		// Subsequent calls on the same day: must NOT award points
		for i := 1; i < numCalls; i++ {
			result, err := svc.AwardVideoPoints(ctx, AwardVideoPointsCommand{
				StudentID:   studentID,
				LessonID:    lessonID,
				SourceTitle: "Lesson 1",
			})
			if err != nil {
				t.Fatalf("call %d AwardVideoPoints failed: %v", i+1, err)
			}
			if result.Awarded {
				t.Fatalf("call %d on same day must NOT award points (Requirement 17.2), but Awarded=true", i+1)
			}
		}

		// Property: exactly one event was recorded for this student+lesson today
		today := time.Now().UTC()
		todayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
		todayEnd := todayStart.Add(24 * time.Hour)

		eventsToday := 0
		for _, e := range deps.eventRepo.events {
			if e.StudentID == studentID &&
				e.SourceID == lessonID &&
				e.Type == domainpoints.PointEventTypeVideoComplete &&
				!e.EarnedAt.Before(todayStart) &&
				e.EarnedAt.Before(todayEnd) {
				eventsToday++
			}
		}
		if eventsToday != 1 {
			t.Fatalf("expected exactly 1 video_complete event today, got %d", eventsToday)
		}

		// Property: calling for a DIFFERENT lesson on the same day awards points
		differentLessonID := uuid.New()
		diffResult, err := svc.AwardVideoPoints(ctx, AwardVideoPointsCommand{
			StudentID:   studentID,
			LessonID:    differentLessonID,
			SourceTitle: "Lesson 2",
		})
		if err != nil {
			t.Fatalf("AwardVideoPoints for different lesson failed: %v", err)
		}
		if !diffResult.Awarded {
			t.Fatal("different lesson on same day must award points")
		}

		// Property: calling for a DIFFERENT student on the same lesson awards points
		differentStudentID := uuid.New()
		diffStudentResult, err := svc.AwardVideoPoints(ctx, AwardVideoPointsCommand{
			StudentID:   differentStudentID,
			LessonID:    lessonID,
			SourceTitle: "Lesson 1",
		})
		if err != nil {
			t.Fatalf("AwardVideoPoints for different student failed: %v", err)
		}
		if !diffStudentResult.Awarded {
			t.Fatal("same lesson for different student must award points")
		}

		// Property: simulate a "next day" by injecting a past event and checking
		// that a fresh repo (representing tomorrow) would award again.
		// We do this by creating a new deps with an event from yesterday.
		deps2 := newPointsPropDeps()
		svc2 := deps2.service()

		// Seed yesterday's event directly into the repo
		yesterday := time.Now().UTC().AddDate(0, 0, -1)
		deps2.eventRepo.events = append(deps2.eventRepo.events, &domainpoints.PointEvent{
			ID:        uuid.New(),
			StudentID: studentID,
			Type:      domainpoints.PointEventTypeVideoComplete,
			SourceID:  lessonID,
			Points:    10,
			EarnedAt:  yesterday,
		})

		// Today's call should award again (Requirement 17.3)
		nextDayResult, err := svc2.AwardVideoPoints(ctx, AwardVideoPointsCommand{
			StudentID:   studentID,
			LessonID:    lessonID,
			SourceTitle: "Lesson 1",
		})
		if err != nil {
			t.Fatalf("AwardVideoPoints on next day failed: %v", err)
		}
		if !nextDayResult.Awarded {
			t.Fatal("same lesson on a different calendar day must award points again (Requirement 17.3)")
		}
	})
}

// ─── Property 49 ─────────────────────────────────────────────────────────────

// TestProperty49_OptedOutStudentsAppearAsAnonymousInLeaderboard verifies that
// students who have opted out of the leaderboard appear as "Anonymous" with a
// nil UUID to other users, but still see their own real name.
//
// **Validates: Requirements 17.11**
func TestProperty49_OptedOutStudentsAppearAsAnonymousInLeaderboard(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		deps := newPointsPropDeps()
		svc := deps.service()
		ctx := context.Background()

		// Generate 2–6 students total
		numStudents := rapid.IntRange(2, 6).Draw(t, "num_students")

		// Generate how many of them opt out (at least 1, at most numStudents-1
		// so there's always at least one non-opted-out student)
		numOptedOut := rapid.IntRange(1, numStudents-1).Draw(t, "num_opted_out")

		studentIDs := make([]uuid.UUID, numStudents)
		studentNames := make([]string, numStudents)
		for i := range studentIDs {
			studentIDs[i] = uuid.New()
			studentNames[i] = "Student" + string(rune('A'+i))
			deps.nameResolver.names[studentIDs[i]] = studentNames[i]
		}

		// Mark the first numOptedOut students as opted out
		optedOutSet := make(map[uuid.UUID]bool)
		for i := 0; i < numOptedOut; i++ {
			deps.leaderboard.optedOut[studentIDs[i]] = true
			optedOutSet[studentIDs[i]] = true
		}

		// Add all students to the leaderboard with distinct scores
		leaderboardKey := "leaderboard:alltime"
		for i, id := range studentIDs {
			score := float64((numStudents - i) * 100) // descending scores
			deps.leaderboard.scores[leaderboardKey] = map[uuid.UUID]float64{}
			_ = score
			_ = id
		}
		// Re-populate properly
		deps.leaderboard.scores[leaderboardKey] = make(map[uuid.UUID]float64)
		for i, id := range studentIDs {
			deps.leaderboard.scores[leaderboardKey][id] = float64((numStudents - i) * 100)
		}

		// Pick a requester who is NOT opted out (last student)
		requesterID := studentIDs[numStudents-1]

		resp, err := svc.GetLeaderboard(ctx, GetLeaderboardCommand{
			RequesterID: requesterID,
			Period:      "alltime",
			Limit:       numStudents,
		})
		if err != nil {
			t.Fatalf("GetLeaderboard failed: %v", err)
		}

		if len(resp.Entries) == 0 {
			t.Fatal("expected leaderboard entries")
		}

		// Property: opted-out students appear as "Anonymous" with uuid.Nil to other users
		for _, entry := range resp.Entries {
			// Determine if this entry corresponds to an opted-out student.
			// Since opted-out students have their ID masked to uuid.Nil,
			// we check entries that are NOT the requester's own entry.
			if entry.StudentID == uuid.Nil {
				// This must be an opted-out student (masked)
				if entry.DisplayName != "Anonymous" {
					t.Fatalf("opted-out student must appear as 'Anonymous', got %q", entry.DisplayName)
				}
			} else if entry.StudentID == requesterID {
				// Requester sees their own real name regardless of opt-out status
				if entry.DisplayName == "Anonymous" {
					t.Fatalf("requester must see their own real name, not 'Anonymous'")
				}
			} else {
				// Non-opted-out student: must show real name
				if optedOutSet[entry.StudentID] {
					t.Fatalf("opted-out student %v should have masked ID but appears with real ID", entry.StudentID)
				}
				if entry.DisplayName == "Anonymous" {
					t.Fatalf("non-opted-out student must show real name, not 'Anonymous'")
				}
			}
		}

		// Property: count of anonymous entries equals numOptedOut
		// (since the requester is not opted out, all opted-out students are masked)
		anonymousCount := 0
		for _, entry := range resp.Entries {
			if entry.StudentID == uuid.Nil && entry.DisplayName == "Anonymous" {
				anonymousCount++
			}
		}
		if anonymousCount != numOptedOut {
			t.Fatalf("expected %d anonymous entries, got %d", numOptedOut, anonymousCount)
		}

		// Property: when the requester IS an opted-out student, they see their own real name
		// Pick an opted-out student as the requester
		optedOutRequesterID := studentIDs[0] // first student is opted out

		resp2, err := svc.GetLeaderboard(ctx, GetLeaderboardCommand{
			RequesterID: optedOutRequesterID,
			Period:      "alltime",
			Limit:       numStudents,
		})
		if err != nil {
			t.Fatalf("GetLeaderboard (opted-out requester) failed: %v", err)
		}

		// The opted-out requester must see their own real name
		for _, entry := range resp2.Entries {
			if entry.StudentID == optedOutRequesterID {
				if entry.DisplayName == "Anonymous" {
					t.Fatal("opted-out student must see their own real name in their own leaderboard view (Requirement 17.11)")
				}
				break
			}
		}

		// Property: other opted-out students (not the requester) still appear as Anonymous
		for _, entry := range resp2.Entries {
			if entry.StudentID == uuid.Nil {
				if entry.DisplayName != "Anonymous" {
					t.Fatalf("other opted-out students must appear as 'Anonymous', got %q", entry.DisplayName)
				}
			}
		}
	})
}
