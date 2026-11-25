package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

// StudentNameResolver implements application/points.StudentNameResolver
// by looking up the user's full_name from the users table.
type StudentNameResolver struct {
	db *sql.DB
}

// NewStudentNameResolver creates a new StudentNameResolver.
func NewStudentNameResolver(db *sql.DB) *StudentNameResolver {
	return &StudentNameResolver{db: db}
}

// GetDisplayName returns the full_name for the given student ID.
func (r *StudentNameResolver) GetDisplayName(ctx context.Context, studentID uuid.UUID) (string, error) {
	var name string
	err := r.db.QueryRowContext(ctx,
		`SELECT full_name FROM users WHERE id = $1 AND deleted_at IS NULL`, studentID,
	).Scan(&name)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return name, err
}
