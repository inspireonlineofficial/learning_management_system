package points

import (
	"time"

	"github.com/google/uuid"
)

// PointEventResponse represents a single point-earning event in the history log.
type PointEventResponse struct {
	ID          uuid.UUID `json:"id"`
	Type        string    `json:"type"`
	SourceID    uuid.UUID `json:"source_id"`
	SourceTitle string    `json:"source_title"`
	Points      int       `json:"points"`
	BonusPoints int       `json:"bonus_points"`
	EarnedAt    time.Time `json:"earned_at"`
}

// DailyBreakdownEntry represents points earned in a single event for today's breakdown.
type DailyBreakdownEntry struct {
	Type        string    `json:"type"`
	SourceTitle string    `json:"source_title"`
	Points      int       `json:"points"`
	BonusPoints int       `json:"bonus_points"`
	EarnedAt    time.Time `json:"earned_at"`
}

type DailyEntry struct {
	Date   string `json:"date"`
	Points int    `json:"points"`
}

type SourceEntry struct {
	Source string `json:"source"`
	Points int    `json:"points"`
}

type MilestoneEntry struct {
	ID         string     `json:"id"`
	Label      string     `json:"label"`
	Threshold  int        `json:"threshold"`
	AchievedAt *time.Time `json:"achieved_at,omitempty"`
}

type RecentEventEntry struct {
	ID        string    `json:"id"`
	Reason    string    `json:"reason"`
	Points    int       `json:"points"`
	CreatedAt time.Time `json:"created_at"`
}

// StudentPointsResponse is the response for GetStudentPoints.
type StudentPointsResponse struct {
	TotalPoints         int                   `json:"total_points"`
	PointsToday         int                   `json:"points_today"`
	PointsThisWeek      int                   `json:"points_this_week"`
	DailyBreakdownToday []DailyBreakdownEntry `json:"daily_breakdown_today"`
	GlobalRank          int                   `json:"global_rank"`
	WeeklyRank          int                   `json:"weekly_rank"`

	// Compatibility fields for frontend PointsBreakdown
	Total             int                `json:"total"`
	StreakDays        int                `json:"streak_days"`
	LongestStreakDays int                `json:"longest_streak_days,omitempty"`
	ThisWeek          int                `json:"this_week"`
	ThisMonth         int                `json:"this_month"`
	Daily             []DailyEntry       `json:"daily"`
	BySource          []SourceEntry      `json:"by_source,omitempty"`
	Milestones        []MilestoneEntry   `json:"milestones,omitempty"`
	RecentEvents      []RecentEventEntry `json:"recent_events"`
}

// PointsHistoryResponse is the paginated response for GetPointsHistory.
type PointsHistoryResponse struct {
	Events []PointEventResponse `json:"events"`
	Meta   PaginationMeta       `json:"meta"`
}

// PaginationMeta holds pagination metadata.
type PaginationMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// PointsConfigResponse is the response after updating the points configuration.
type PointsConfigResponse struct {
	PointsPerVideo          int `json:"points_per_video"`
	PointsPerQuizPass       int `json:"points_per_quiz_pass"`
	BonusPointsPerfectScore int `json:"bonus_points_perfect_score"`
}

// AwardPointsResult is returned by AwardVideoPoints / AwardQuizPoints.
// Awarded is false when the dedup check prevented awarding.
type AwardPointsResult struct {
	Awarded     bool
	Points      int
	BonusPoints int
}

// LeaderboardEntry represents a single ranked student on the leaderboard.
// When the student has opted out and the requester is not that student,
// DisplayName is "Anonymous" and StudentID is masked (zero UUID).
type LeaderboardEntry struct {
	Rank        int       `json:"rank"`
	StudentID   uuid.UUID `json:"student_id"`
	DisplayName string    `json:"display_name"`
	Score       float64   `json:"score"`
}

// LeaderboardResponse is the response for GetLeaderboard.
type LeaderboardResponse struct {
	Period  string             `json:"period"`
	Entries []LeaderboardEntry `json:"entries"`
}

// ToggleLeaderboardOptOutResponse confirms the new opt-out state.
type ToggleLeaderboardOptOutResponse struct {
	OptedOut bool `json:"opted_out"`
}
