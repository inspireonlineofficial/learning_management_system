package points

import (
	"context"
	"fmt"
	"sort"
	"time"

	"lms-backend/internal/domain/points"
	"lms-backend/pkg/apperrors"
	"lms-backend/pkg/logger"

	"github.com/google/uuid"
)

// Service defines the interface for the Points engine use cases.
type Service interface {
	// AwardVideoPoints awards points_per_video once per lesson per UTC calendar day.
	// Requirements: 17.1, 17.2, 17.3
	AwardVideoPoints(ctx context.Context, cmd AwardVideoPointsCommand) (*AwardPointsResult, error)

	// AwardQuizPoints awards points_per_quiz_pass on first pass and
	// bonus_points_perfect_score on first perfect score.
	// Requirements: 17.4, 17.5, 17.6
	AwardQuizPoints(ctx context.Context, cmd AwardQuizPointsCommand) (*AwardPointsResult, error)

	// GetStudentPoints returns aggregated point totals and ranks.
	// Requirements: 17.7
	GetStudentPoints(ctx context.Context, cmd GetStudentPointsCommand) (*StudentPointsResponse, error)

	// GetPointsHistory returns a paginated log of point events.
	// Requirements: 17.8
	GetPointsHistory(ctx context.Context, cmd GetPointsHistoryCommand) (*PointsHistoryResponse, error)

	// UpdatePointsConfig (admin) updates the platform-wide config and records an audit log.
	// Requirements: 17.9
	UpdatePointsConfig(ctx context.Context, cmd UpdatePointsConfigCommand) (*PointsConfigResponse, error)

	// GetPointsConfig returns the current platform-wide config for admin settings.
	GetPointsConfig(ctx context.Context) (*PointsConfigResponse, error)

	// GetLeaderboard returns the top-N ranked students for weekly or alltime periods.
	// Opted-out students appear as "Anonymous" to other users.
	// Requirements: 17.10, 17.11
	GetLeaderboard(ctx context.Context, cmd GetLeaderboardCommand) (*LeaderboardResponse, error)

	// ToggleLeaderboardOptOut allows a student to opt in or out of the public leaderboard.
	// Requirements: 17.11
	ToggleLeaderboardOptOut(ctx context.Context, cmd ToggleLeaderboardOptOutCommand) (*ToggleLeaderboardOptOutResponse, error)
}

// AuditLogger records privileged admin actions.
type AuditLogger interface {
	LogAction(ctx context.Context, actorID uuid.UUID, actorName, action, targetType string, targetID uuid.UUID, metadata map[string]interface{}, ipAddress string) error
}

// LeaderboardRawEntry is the raw entry returned by the LeaderboardStore before
// opt-out masking is applied.
type LeaderboardRawEntry struct {
	MemberID uuid.UUID
	Score    float64
}

// LeaderboardStore defines the Redis sorted-set operations for leaderboards.
// The infrastructure layer (Redis) implements this interface.
type LeaderboardStore interface {
	// AddScore increments (or sets) the score for a member in the given sorted-set key.
	AddScore(ctx context.Context, key string, memberID uuid.UUID, score float64) error
	// GetTopN returns the top N members with their scores in descending order.
	GetTopN(ctx context.Context, key string, n int) ([]LeaderboardRawEntry, error)
	// ResetWeekly deletes the weekly leaderboard sorted set (Sunday midnight UTC job).
	ResetWeekly(ctx context.Context) error
	// GetOptOutStatus returns true if the student has opted out of the leaderboard.
	GetOptOutStatus(ctx context.Context, studentID uuid.UUID) (bool, error)
	// SetOptOutStatus persists the student's opt-out preference.
	SetOptOutStatus(ctx context.Context, studentID uuid.UUID, optOut bool) error
}

// StudentNameResolver resolves a student's display name from their ID.
// Typically backed by the users repository.
type StudentNameResolver interface {
	GetDisplayName(ctx context.Context, studentID uuid.UUID) (string, error)
}

// PointsRankRepository provides rank queries that are not part of the core
// PointEventRepository (e.g. counting students with more points).
type PointsRankRepository interface {
	// CountStudentsWithMoreTotalPoints returns the number of students whose
	// total points exceed the given value. Rank = count + 1.
	CountStudentsWithMoreTotalPoints(ctx context.Context, totalPoints int) (int, error)
	// CountStudentsWithMoreWeeklyPoints returns the number of students whose
	// weekly points (since weekStart) exceed the given value.
	CountStudentsWithMoreWeeklyPoints(ctx context.Context, weeklyPoints int, weekStart time.Time) (int, error)
	// SumByStudentIDForDay returns the total points earned by a student on a
	// specific UTC calendar day (used for daily_breakdown_today).
	FindEventsForDay(ctx context.Context, studentID uuid.UUID, day time.Time) ([]*points.PointEvent, error)
}

type service struct {
	eventRepo    points.PointEventRepository
	configRepo   points.PointsConfigRepository
	rankRepo     PointsRankRepository
	leaderboard  LeaderboardStore
	nameResolver StudentNameResolver
	audit        AuditLogger
}

// NewService creates a new Points engine service.
func NewService(
	eventRepo points.PointEventRepository,
	configRepo points.PointsConfigRepository,
	rankRepo PointsRankRepository,
	leaderboard LeaderboardStore,
	nameResolver StudentNameResolver,
	audit AuditLogger,
) Service {
	return &service{
		eventRepo:    eventRepo,
		configRepo:   configRepo,
		rankRepo:     rankRepo,
		leaderboard:  leaderboard,
		nameResolver: nameResolver,
		audit:        audit,
	}
}

// AwardVideoPoints awards points_per_video once per lesson per UTC calendar day.
// Requirements: 17.1, 17.2, 17.3
func (s *service) AwardVideoPoints(ctx context.Context, cmd AwardVideoPointsCommand) (*AwardPointsResult, error) {
	// Load current config
	cfg, err := s.configRepo.Get(ctx)
	if err != nil {
		return nil, apperrors.NewInternalError("CONFIG_LOAD_FAILED", "failed to load points configuration")
	}

	// Daily dedup: truncate to UTC midnight to get the calendar day boundary.
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	exists, err := s.eventRepo.ExistsForSourceOnDay(ctx, cmd.StudentID, cmd.LessonID, points.PointEventTypeVideoComplete, today)
	if err != nil {
		return nil, apperrors.NewInternalError("DEDUP_CHECK_FAILED", "failed to check daily dedup")
	}
	if exists {
		// Already awarded today — do not re-award (Requirement 17.2)
		return &AwardPointsResult{Awarded: false}, nil
	}

	// Create the point event
	event := &points.PointEvent{
		ID:          uuid.New(),
		StudentID:   cmd.StudentID,
		Type:        points.PointEventTypeVideoComplete,
		SourceID:    cmd.LessonID,
		SourceTitle: cmd.SourceTitle,
		Points:      cfg.PointsPerVideo,
		BonusPoints: 0,
		EarnedAt:    now,
	}

	if err := s.eventRepo.Create(ctx, event); err != nil {
		return nil, apperrors.NewInternalError("EVENT_CREATE_FAILED", "failed to record point event")
	}

	logger.Info(ctx, "Video points awarded",
		"student_id", cmd.StudentID,
		"lesson_id", cmd.LessonID,
		"points", cfg.PointsPerVideo,
	)

	return &AwardPointsResult{
		Awarded: true,
		Points:  cfg.PointsPerVideo,
	}, nil
}

// AwardQuizPoints awards points on first pass and bonus on first perfect score.
// Requirements: 17.4, 17.5, 17.6
func (s *service) AwardQuizPoints(ctx context.Context, cmd AwardQuizPointsCommand) (*AwardPointsResult, error) {
	// Load current config
	cfg, err := s.configRepo.Get(ctx)
	if err != nil {
		return nil, apperrors.NewInternalError("CONFIG_LOAD_FAILED", "failed to load points configuration")
	}

	// Check if a quiz_pass event already exists for this student+quiz (Requirement 17.6)
	alreadyPassed, err := s.eventRepo.ExistsPassingForSource(ctx, cmd.StudentID, cmd.QuizID, points.PointEventTypeQuizPass)
	if err != nil {
		return nil, apperrors.NewInternalError("DEDUP_CHECK_FAILED", "failed to check quiz pass dedup")
	}
	if alreadyPassed {
		// Already awarded for a previous pass — do not re-award
		return &AwardPointsResult{Awarded: false}, nil
	}

	now := time.Now().UTC()
	basePoints := cfg.PointsPerQuizPass
	bonusPoints := 0

	// Check for first perfect score (Requirement 17.5)
	isPerfect := cmd.ScorePercent == 100.0
	if isPerfect {
		// Check if a quiz_perfect event already exists
		alreadyPerfect, err := s.eventRepo.ExistsPassingForSource(ctx, cmd.StudentID, cmd.QuizID, points.PointEventTypeQuizPerfect)
		if err != nil {
			return nil, apperrors.NewInternalError("DEDUP_CHECK_FAILED", "failed to check perfect score dedup")
		}
		if !alreadyPerfect {
			bonusPoints = cfg.BonusPointsPerfectScore
		}
	}

	// Record the quiz_pass event
	passEvent := &points.PointEvent{
		ID:          uuid.New(),
		StudentID:   cmd.StudentID,
		Type:        points.PointEventTypeQuizPass,
		SourceID:    cmd.QuizID,
		SourceTitle: cmd.SourceTitle,
		Points:      basePoints,
		BonusPoints: 0,
		EarnedAt:    now,
	}
	if err := s.eventRepo.Create(ctx, passEvent); err != nil {
		return nil, apperrors.NewInternalError("EVENT_CREATE_FAILED", "failed to record quiz pass event")
	}

	// Record a separate quiz_perfect event when bonus applies
	if bonusPoints > 0 {
		perfectEvent := &points.PointEvent{
			ID:          uuid.New(),
			StudentID:   cmd.StudentID,
			Type:        points.PointEventTypeQuizPerfect,
			SourceID:    cmd.QuizID,
			SourceTitle: cmd.SourceTitle,
			Points:      0,
			BonusPoints: bonusPoints,
			EarnedAt:    now,
		}
		if err := s.eventRepo.Create(ctx, perfectEvent); err != nil {
			return nil, apperrors.NewInternalError("EVENT_CREATE_FAILED", "failed to record perfect score event")
		}
	}

	logger.Info(ctx, "Quiz points awarded",
		"student_id", cmd.StudentID,
		"quiz_id", cmd.QuizID,
		"points", basePoints,
		"bonus_points", bonusPoints,
	)

	return &AwardPointsResult{
		Awarded:     true,
		Points:      basePoints,
		BonusPoints: bonusPoints,
	}, nil
}

// GetStudentPoints returns aggregated totals and leaderboard ranks.
// Requirements: 17.7
func (s *service) GetStudentPoints(ctx context.Context, cmd GetStudentPointsCommand) (*StudentPointsResponse, error) {
	now := time.Now().UTC()

	// Total points (all time)
	totalPoints, err := s.eventRepo.SumByStudentID(ctx, cmd.StudentID)
	if err != nil {
		return nil, apperrors.NewInternalError("POINTS_QUERY_FAILED", "failed to query total points")
	}

	// Points today (since UTC midnight)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	pointsToday, err := s.eventRepo.SumByStudentIDSince(ctx, cmd.StudentID, todayStart)
	if err != nil {
		return nil, apperrors.NewInternalError("POINTS_QUERY_FAILED", "failed to query today's points")
	}

	// Points this week (since last Monday UTC midnight)
	weekStart := startOfWeek(now)
	pointsThisWeek, err := s.eventRepo.SumByStudentIDSince(ctx, cmd.StudentID, weekStart)
	if err != nil {
		return nil, apperrors.NewInternalError("POINTS_QUERY_FAILED", "failed to query weekly points")
	}

	// Points this month (since start of calendar month UTC midnight)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	pointsThisMonth, err := s.eventRepo.SumByStudentIDSince(ctx, cmd.StudentID, monthStart)
	if err != nil {
		pointsThisMonth = 0 // non-fatal fallback
	}

	// Daily breakdown for today
	todayEvents, err := s.rankRepo.FindEventsForDay(ctx, cmd.StudentID, todayStart)
	if err != nil {
		return nil, apperrors.NewInternalError("POINTS_QUERY_FAILED", "failed to query today's events")
	}
	breakdown := make([]DailyBreakdownEntry, 0, len(todayEvents))
	for _, e := range todayEvents {
		breakdown = append(breakdown, DailyBreakdownEntry{
			Type:        string(e.Type),
			SourceTitle: e.SourceTitle,
			Points:      e.Points,
			BonusPoints: e.BonusPoints,
			EarnedAt:    e.EarnedAt,
		})
	}

	// Global rank: count students with more total points + 1
	studentsAhead, err := s.rankRepo.CountStudentsWithMoreTotalPoints(ctx, totalPoints)
	if err != nil {
		return nil, apperrors.NewInternalError("RANK_QUERY_FAILED", "failed to query global rank")
	}
	globalRank := studentsAhead + 1

	// Weekly rank: count students with more weekly points + 1
	weeklyAhead, err := s.rankRepo.CountStudentsWithMoreWeeklyPoints(ctx, pointsThisWeek, weekStart)
	if err != nil {
		return nil, apperrors.NewInternalError("RANK_QUERY_FAILED", "failed to query weekly rank")
	}
	weeklyRank := weeklyAhead + 1

	// Fetch up to 1000 events to compute daily breakdown chart, bySource, streaks, and recent events
	events, _, err := s.eventRepo.FindByStudentID(ctx, cmd.StudentID, 1, 1000)
	if err != nil {
		events = nil // non-fatal fallback
	}

	uniqueDatesMap := make(map[string]int)
	sourceMap := make(map[string]int)
	var recentEvents []RecentEventEntry

	for _, e := range events {
		dStr := e.EarnedAt.UTC().Format("2006-01-02")
		pts := e.Points + e.BonusPoints
		uniqueDatesMap[dStr] += pts

		src := "other"
		if e.Type == points.PointEventTypeVideoComplete {
			src = "video"
		} else if e.Type == points.PointEventTypeQuizPass || e.Type == points.PointEventTypeQuizPerfect {
			src = "quiz"
		}
		sourceMap[src] += pts

		if len(recentEvents) < 10 {
			reason := e.SourceTitle
			if reason == "" {
				if e.Type == points.PointEventTypeVideoComplete {
					reason = "Watched video"
				} else if e.Type == points.PointEventTypeQuizPass {
					reason = "Passed quiz"
				} else if e.Type == points.PointEventTypeQuizPerfect {
					reason = "Perfect score bonus"
				} else {
					reason = "Points earned"
				}
			} else {
				if e.Type == points.PointEventTypeVideoComplete {
					reason = fmt.Sprintf("Completed video: %s", e.SourceTitle)
				} else if e.Type == points.PointEventTypeQuizPass {
					reason = fmt.Sprintf("Passed quiz: %s", e.SourceTitle)
				} else if e.Type == points.PointEventTypeQuizPerfect {
					reason = fmt.Sprintf("Perfect score on: %s", e.SourceTitle)
				}
			}
			recentEvents = append(recentEvents, RecentEventEntry{
				ID:        e.ID.String(),
				Reason:    reason,
				Points:    pts,
				CreatedAt: e.EarnedAt,
			})
		}
	}

	var dates []time.Time
	for dStr := range uniqueDatesMap {
		if t, err := time.Parse("2006-01-02", dStr); err == nil {
			dates = append(dates, t)
		}
	}
	sort.Slice(dates, func(i, j int) bool {
		return dates[i].Before(dates[j])
	})

	longestStreak := 0
	currentRun := 0
	var prevDate time.Time

	for i, d := range dates {
		if i == 0 {
			currentRun = 1
		} else {
			nextDay := prevDate.AddDate(0, 0, 1)
			if nextDay.Equal(d) {
				currentRun++
			} else if d.Equal(prevDate) {
				// same day, ignore
			} else {
				if currentRun > longestStreak {
					longestStreak = currentRun
				}
				currentRun = 1
			}
		}
		prevDate = d
	}
	if currentRun > longestStreak {
		longestStreak = currentRun
	}

	currentStreak := 0
	if len(dates) > 0 {
		lastDate := dates[len(dates)-1]
		today := time.Now().UTC().Truncate(24 * time.Hour)
		yesterday := today.AddDate(0, 0, -1)
		if lastDate.Equal(today) || lastDate.Equal(yesterday) {
			currentStreak = currentRun
		}
	}

	var dailyList []DailyEntry
	for dStr, pts := range uniqueDatesMap {
		dailyList = append(dailyList, DailyEntry{
			Date:   dStr,
			Points: pts,
		})
	}
	sort.Slice(dailyList, func(i, j int) bool {
		return dailyList[i].Date < dailyList[j].Date
	})

	// If a period is specified, filter the daily list. e.g. "7d"
	if cmd.Period == "7d" && len(dailyList) > 7 {
		dailyList = dailyList[len(dailyList)-7:]
	} else if cmd.Period == "30d" && len(dailyList) > 30 {
		dailyList = dailyList[len(dailyList)-30:]
	}

	var bySourceList []SourceEntry
	for src, pts := range sourceMap {
		bySourceList = append(bySourceList, SourceEntry{
			Source: src,
			Points: pts,
		})
	}

	thresholds := []struct {
		id    string
		label string
		val   int
	}{
		{"m1", "First Steps", 100},
		{"m2", "Knowledge Seeker", 500},
		{"m3", "Point Master", 1000},
		{"m4", "Elite Scholar", 5000},
	}
	var milestones []MilestoneEntry
	for _, m := range thresholds {
		var achievedAt *time.Time
		if totalPoints >= m.val {
			cum := 0
			for i := len(events) - 1; i >= 0; i-- {
				e := events[i]
				cum += e.Points + e.BonusPoints
				if cum >= m.val {
					achievedAt = &e.EarnedAt
					break
				}
			}
			if achievedAt == nil && len(events) > 0 {
				achievedAt = &events[len(events)-1].EarnedAt
			}
		}
		milestones = append(milestones, MilestoneEntry{
			ID:         m.id,
			Label:      m.label,
			Threshold:  m.val,
			AchievedAt: achievedAt,
		})
	}

	return &StudentPointsResponse{
		TotalPoints:         totalPoints,
		PointsToday:         pointsToday,
		PointsThisWeek:      pointsThisWeek,
		DailyBreakdownToday: breakdown,
		GlobalRank:          globalRank,
		WeeklyRank:          weeklyRank,

		// Compatibility fields
		Total:             totalPoints,
		StreakDays:        currentStreak,
		LongestStreakDays: longestStreak,
		ThisWeek:          pointsThisWeek,
		ThisMonth:         pointsThisMonth,
		Daily:             dailyList,
		BySource:          bySourceList,
		Milestones:        milestones,
		RecentEvents:      recentEvents,
	}, nil
}

// GetPointsHistory returns a paginated log of point events.
// Requirements: 17.8
func (s *service) GetPointsHistory(ctx context.Context, cmd GetPointsHistoryCommand) (*PointsHistoryResponse, error) {
	if cmd.Page < 1 {
		cmd.Page = 1
	}
	if cmd.Limit < 1 || cmd.Limit > 100 {
		cmd.Limit = 20
	}

	events, total, err := s.eventRepo.FindByStudentID(ctx, cmd.StudentID, cmd.Page, cmd.Limit)
	if err != nil {
		return nil, apperrors.NewInternalError("HISTORY_QUERY_FAILED", "failed to query points history")
	}

	responses := make([]PointEventResponse, 0, len(events))
	for _, e := range events {
		responses = append(responses, PointEventResponse{
			ID:          e.ID,
			Type:        string(e.Type),
			SourceID:    e.SourceID,
			SourceTitle: e.SourceTitle,
			Points:      e.Points,
			BonusPoints: e.BonusPoints,
			EarnedAt:    e.EarnedAt,
		})
	}

	totalPages := (total + cmd.Limit - 1) / cmd.Limit
	if totalPages < 1 {
		totalPages = 1
	}

	return &PointsHistoryResponse{
		Events: responses,
		Meta: PaginationMeta{
			Page:       cmd.Page,
			Limit:      cmd.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// GetPointsConfig returns the current platform-wide points configuration.
func (s *service) GetPointsConfig(ctx context.Context) (*PointsConfigResponse, error) {
	cfg, err := s.configRepo.Get(ctx)
	if err != nil {
		return nil, apperrors.NewInternalError("CONFIG_LOAD_FAILED", "failed to load points configuration")
	}
	return toPointsConfigResponse(cfg), nil
}

// UpdatePointsConfig updates the platform-wide points config and records an audit log.
// Requirements: 17.9
func (s *service) UpdatePointsConfig(ctx context.Context, cmd UpdatePointsConfigCommand) (*PointsConfigResponse, error) {
	cfg, err := s.configRepo.Get(ctx)
	if err != nil {
		return nil, apperrors.NewInternalError("CONFIG_LOAD_FAILED", "failed to load points configuration")
	}

	// Apply partial updates
	if cmd.PointsPerVideo != nil {
		cfg.PointsPerVideo = *cmd.PointsPerVideo
	}
	if cmd.PointsPerQuizPass != nil {
		cfg.PointsPerQuizPass = *cmd.PointsPerQuizPass
	}
	if cmd.BonusPointsPerfectScore != nil {
		cfg.BonusPointsPerfectScore = *cmd.BonusPointsPerfectScore
	}

	now := time.Now().UTC()
	cfg.UpdatedAt = &now
	cfg.UpdatedBy = &cmd.ActorID

	if err := s.configRepo.Update(ctx, cfg); err != nil {
		return nil, apperrors.NewInternalError("CONFIG_UPDATE_FAILED", "failed to update points configuration")
	}

	// Audit log — action: points_config_changed (Requirement 9.4, 17.9)
	metadata := map[string]interface{}{
		"points_per_video":           cfg.PointsPerVideo,
		"points_per_quiz_pass":       cfg.PointsPerQuizPass,
		"bonus_points_perfect_score": cfg.BonusPointsPerfectScore,
	}
	// Use a zero UUID as target since PointsConfig is a singleton (id=1, not a UUID)
	if err := s.audit.LogAction(ctx, cmd.ActorID, cmd.ActorName, "points_config_changed", "points_config", uuid.Nil, metadata, cmd.IPAddress); err != nil {
		logger.Error(ctx, "Failed to record audit log for points config change", "error", err)
		// Non-fatal: config was already updated successfully
	}

	logger.Info(ctx, "Points config updated",
		"actor_id", cmd.ActorID,
		"points_per_video", cfg.PointsPerVideo,
		"points_per_quiz_pass", cfg.PointsPerQuizPass,
		"bonus_points_perfect_score", cfg.BonusPointsPerfectScore,
	)

	return toPointsConfigResponse(cfg), nil
}

func toPointsConfigResponse(cfg *points.PointsConfig) *PointsConfigResponse {
	return &PointsConfigResponse{
		PointsPerVideo:          cfg.PointsPerVideo,
		PointsPerQuizPass:       cfg.PointsPerQuizPass,
		BonusPointsPerfectScore: cfg.BonusPointsPerfectScore,
	}
}

// startOfWeek returns the UTC midnight of the most recent Monday.
func startOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday → treat as day 7 so Monday is day 1
	}
	daysBack := weekday - 1
	monday := t.AddDate(0, 0, -daysBack)
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
}

// leaderboardKey returns the Redis sorted-set key for the given period.
func leaderboardKey(period string) string {
	switch period {
	case "weekly":
		return "leaderboard:weekly"
	default:
		return "leaderboard:alltime"
	}
}

// GetLeaderboard returns the top-N ranked students for weekly or alltime periods.
// Opted-out students appear as "Anonymous" (with a masked zero UUID) to other users.
// Requirements: 17.10, 17.11
func (s *service) GetLeaderboard(ctx context.Context, cmd GetLeaderboardCommand) (*LeaderboardResponse, error) {
	limit := cmd.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	period := cmd.Period
	if period != "weekly" && period != "alltime" {
		period = "alltime"
	}

	key := leaderboardKey(period)

	rawEntries, err := s.leaderboard.GetTopN(ctx, key, limit)
	if err != nil {
		return nil, apperrors.NewInternalError("LEADERBOARD_QUERY_FAILED", "failed to query leaderboard")
	}

	entries := make([]LeaderboardEntry, 0, len(rawEntries))
	for i, raw := range rawEntries {
		// Resolve display name
		displayName, err := s.nameResolver.GetDisplayName(ctx, raw.MemberID)
		if err != nil {
			// Non-fatal: fall back to empty string rather than failing the whole request
			logger.Error(ctx, "Failed to resolve student display name", "student_id", raw.MemberID, "error", err)
			displayName = ""
		}

		studentID := raw.MemberID

		// Opt-out masking: if the student opted out and the requester is not that student,
		// replace their name with "Anonymous" and mask their ID. (Requirement 17.11)
		if raw.MemberID != cmd.RequesterID {
			optedOut, err := s.leaderboard.GetOptOutStatus(ctx, raw.MemberID)
			if err != nil {
				logger.Error(ctx, "Failed to check leaderboard opt-out status", "student_id", raw.MemberID, "error", err)
				// Treat as not opted out on error to avoid hiding data incorrectly
				optedOut = false
			}
			if optedOut {
				displayName = "Anonymous"
				studentID = uuid.Nil
			}
		}

		entries = append(entries, LeaderboardEntry{
			Rank:        i + 1,
			StudentID:   studentID,
			DisplayName: displayName,
			Score:       raw.Score,
		})
	}

	return &LeaderboardResponse{
		Period:  period,
		Entries: entries,
	}, nil
}

// ToggleLeaderboardOptOut allows a student to opt in or out of the public leaderboard.
// Requirements: 17.11
func (s *service) ToggleLeaderboardOptOut(ctx context.Context, cmd ToggleLeaderboardOptOutCommand) (*ToggleLeaderboardOptOutResponse, error) {
	if err := s.leaderboard.SetOptOutStatus(ctx, cmd.StudentID, cmd.OptOut); err != nil {
		return nil, apperrors.NewInternalError("OPT_OUT_FAILED", "failed to update leaderboard opt-out preference")
	}

	logger.Info(ctx, "Leaderboard opt-out updated",
		"student_id", cmd.StudentID,
		"opted_out", cmd.OptOut,
	)

	return &ToggleLeaderboardOptOutResponse{OptedOut: cmd.OptOut}, nil
}
