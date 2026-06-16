package points

import (
	"time"

	"github.com/google/uuid"
)

// PointsConfig is a singleton entity holding the gamification point values
type PointsConfig struct {
	ID                      int // always 1
	PointsPerVideo          int
	PointsPerQuizPass       int
	BonusPointsPerfectScore int
	UpdatedAt               *time.Time
	UpdatedBy               *uuid.UUID // FK → users
}
