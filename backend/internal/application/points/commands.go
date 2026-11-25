package points

import "github.com/google/uuid"

// GetLeaderboardCommand requests the leaderboard for a given period.
// Requirements: 17.10, 17.11
type GetLeaderboardCommand struct {
	// RequesterID is the authenticated user making the request.
	// Used to determine whether to show or mask opted-out entries.
	RequesterID uuid.UUID
	// Period is "weekly" or "alltime".
	Period string
	// CourseID optionally scopes the leaderboard to enrolled students of a course.
	CourseID *uuid.UUID
	// Limit is the number of top entries to return (default 20, max 100).
	Limit int
}

// ToggleLeaderboardOptOutCommand allows a student to opt in or out of the leaderboard.
// Requirements: 17.11
type ToggleLeaderboardOptOutCommand struct {
	StudentID uuid.UUID
	// OptOut true = opt out, false = opt in.
	OptOut bool
}

// AwardVideoPointsCommand is issued when a lesson is marked complete.
// The Points engine performs a daily dedup check before awarding.
type AwardVideoPointsCommand struct {
	StudentID   uuid.UUID
	LessonID    uuid.UUID
	SourceTitle string
}

// AwardQuizPointsCommand is issued when a student passes a quiz.
// The Points engine checks first-pass and first-perfect-score dedup.
type AwardQuizPointsCommand struct {
	StudentID    uuid.UUID
	QuizID       uuid.UUID
	SourceTitle  string
	ScorePercent float64 // used to detect perfect score (100.0)
}

// GetStudentPointsCommand requests the aggregated points summary for a student.
type GetStudentPointsCommand struct {
	StudentID uuid.UUID
	Period    string
}

// GetPointsHistoryCommand requests a paginated log of point events.
type GetPointsHistoryCommand struct {
	StudentID uuid.UUID
	Page      int
	Limit     int
}

// UpdatePointsConfigCommand (admin) updates the platform-wide points configuration.
type UpdatePointsConfigCommand struct {
	ActorID                 uuid.UUID
	ActorName               string
	PointsPerVideo          *int
	PointsPerQuizPass       *int
	BonusPointsPerfectScore *int
	IPAddress               string
}
