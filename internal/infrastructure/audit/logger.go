package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Logger implements audit logging
type Logger struct {
	db *sql.DB
}

// NewLogger creates a new audit logger
func NewLogger(db *sql.DB) *Logger {
	return &Logger{db: db}
}

// LogAdminLogin logs an admin login attempt
func (l *Logger) LogAdminLogin(ctx context.Context, userID uuid.UUID, username, ipAddress string, success bool) error {
	action := "admin_login_failed"
	if success {
		action = "admin_login_success"
	}

	query := `
		INSERT INTO audit_logs (id, actor_id, actor_name, action, target_type, target_id, metadata, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	id := uuid.New()
	now := time.Now().UTC()
	metadata := fmt.Sprintf(`{"username": "%s", "success": %t}`, username, success)

	_, err := l.db.ExecContext(ctx, query,
		id, userID, username, action, "user", userID, metadata, ipAddress, now,
	)

	if err != nil {
		return fmt.Errorf("failed to log admin login: %w", err)
	}

	return nil
}

// LogAction logs a generic audit action
func (l *Logger) LogAction(ctx context.Context, actorID uuid.UUID, actorName, action, targetType string, targetID uuid.UUID, metadata map[string]interface{}, ipAddress string) error {
	query := `
		INSERT INTO audit_logs (id, actor_id, actor_name, action, target_type, target_id, metadata, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	id := uuid.New()
	now := time.Now().UTC()

	metadataJSON := []byte("null")
	if metadata != nil && len(metadata) > 0 {
		var err error
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal audit metadata: %w", err)
		}
	}

	_, err := l.db.ExecContext(ctx, query,
		id, actorID, actorName, action, targetType, targetID, metadataJSON, ipAddress, now,
	)

	if err != nil {
		return fmt.Errorf("failed to log action: %w", err)
	}

	return nil
}
