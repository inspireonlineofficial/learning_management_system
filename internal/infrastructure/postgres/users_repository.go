package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"lms-backend/internal/domain/users"
	"time"

	"github.com/google/uuid"
)

// StudentProfileRepository implements users.StudentProfileRepository
type StudentProfileRepository struct {
	db *sql.DB
}

// NewStudentProfileRepository creates a new StudentProfileRepository
func NewStudentProfileRepository(db *sql.DB) *StudentProfileRepository {
	return &StudentProfileRepository{db: db}
}

// Upsert creates or updates a student profile and sets profile_complete atomically
func (r *StudentProfileRepository) Upsert(ctx context.Context, profile *users.StudentProfile) error {
	// Begin transaction to ensure atomicity
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	profile.UpdatedAt = time.Now().UTC()

	// Upsert student profile
	profileQuery := `
		INSERT INTO student_profiles (user_id, school_name, class_grade, roll_number, date_of_birth, gender, guardian_name, guardian_contact, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id) DO UPDATE SET
			school_name = EXCLUDED.school_name,
			class_grade = EXCLUDED.class_grade,
			roll_number = EXCLUDED.roll_number,
			date_of_birth = EXCLUDED.date_of_birth,
			gender = EXCLUDED.gender,
			guardian_name = EXCLUDED.guardian_name,
			guardian_contact = EXCLUDED.guardian_contact,
			updated_at = EXCLUDED.updated_at
	`
	_, err = tx.ExecContext(ctx, profileQuery,
		profile.UserID, profile.SchoolName, profile.ClassGrade, profile.RollNumber,
		profile.DateOfBirth, profile.Gender, profile.GuardianName, profile.GuardianContact,
		profile.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert student profile: %w", err)
	}

	// Set profile_complete = true atomically in the same transaction
	// Only set to true if it's currently false (first submission)
	// Subsequent updates do NOT reset profile_complete
	userQuery := `
		UPDATE users
		SET profile_complete = true, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`
	result, err := tx.ExecContext(ctx, userQuery, profile.UserID, profile.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update profile_complete: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found or already deleted")
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// FindByUserID retrieves a student profile by user ID
func (r *StudentProfileRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*users.StudentProfile, error) {
	query := `
		SELECT user_id, school_name, class_grade, roll_number, date_of_birth, gender, guardian_name, guardian_contact, updated_at
		FROM student_profiles
		WHERE user_id = $1
	`
	profile := &users.StudentProfile{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&profile.UserID, &profile.SchoolName, &profile.ClassGrade, &profile.RollNumber,
		&profile.DateOfBirth, &profile.Gender, &profile.GuardianName, &profile.GuardianContact,
		&profile.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("student profile not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find student profile: %w", err)
	}
	return profile, nil
}

// UpdateByAdmin updates a student profile by admin (for admin operations)
func (r *StudentProfileRepository) UpdateByAdmin(ctx context.Context, profile *users.StudentProfile) error {
	profile.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE student_profiles
		SET school_name = $2, class_grade = $3, roll_number = $4, date_of_birth = $5, 
		    gender = $6, guardian_name = $7, guardian_contact = $8, updated_at = $9
		WHERE user_id = $1
	`
	result, err := r.db.ExecContext(ctx, query,
		profile.UserID, profile.SchoolName, profile.ClassGrade, profile.RollNumber,
		profile.DateOfBirth, profile.Gender, profile.GuardianName, profile.GuardianContact,
		profile.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update student profile: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("student profile not found")
	}

	return nil
}
