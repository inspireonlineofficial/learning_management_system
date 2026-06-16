package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	domainrbac "lms-backend/internal/domain/rbac"

	"github.com/google/uuid"
)

// RBACRepository implements domainrbac.PermissionRepository.
type RBACRepository struct {
	db *sql.DB
}

// NewRBACRepository creates a new RBACRepository.
func NewRBACRepository(db *sql.DB) *RBACRepository {
	return &RBACRepository{db: db}
}

// ListByRole returns all permissions for a given role.
func (r *RBACRepository) ListByRole(ctx context.Context, role string) ([]domainrbac.Permission, error) {
	query := `
		SELECT id, role, resource, action, created_at
		FROM rbac_permissions
		WHERE role = $1
		ORDER BY resource, action
	`
	rows, err := r.db.QueryContext(ctx, query, role)
	if err != nil {
		return nil, fmt.Errorf("list permissions by role: %w", err)
	}
	defer rows.Close()

	var perms []domainrbac.Permission
	for rows.Next() {
		var p domainrbac.Permission
		if err := rows.Scan(&p.ID, &p.Role, &p.Resource, &p.Action, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan permission: %w", err)
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

// ListAll returns all permissions grouped by role.
func (r *RBACRepository) ListAll(ctx context.Context) ([]domainrbac.RolePermissions, error) {
	query := `
		SELECT id, role, resource, action, created_at
		FROM rbac_permissions
		ORDER BY role, resource, action
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list all permissions: %w", err)
	}
	defer rows.Close()

	roleMap := make(map[string]*domainrbac.RolePermissions)
	roleOrder := []string{}

	for rows.Next() {
		var p domainrbac.Permission
		if err := rows.Scan(&p.ID, &p.Role, &p.Resource, &p.Action, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan permission: %w", err)
		}
		if _, exists := roleMap[p.Role]; !exists {
			roleMap[p.Role] = &domainrbac.RolePermissions{Role: p.Role}
			roleOrder = append(roleOrder, p.Role)
		}
		roleMap[p.Role].Permissions = append(roleMap[p.Role].Permissions, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]domainrbac.RolePermissions, 0, len(roleOrder))
	for _, role := range roleOrder {
		result = append(result, *roleMap[role])
	}
	return result, nil
}

// ReplaceForRole atomically replaces all permissions for a role within a transaction.
// Requirements: 9.3
func (r *RBACRepository) ReplaceForRole(ctx context.Context, role string, permissions []domainrbac.Permission) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete existing permissions for this role
	if _, err := tx.ExecContext(ctx, `DELETE FROM rbac_permissions WHERE role = $1`, role); err != nil {
		return fmt.Errorf("delete existing permissions: %w", err)
	}

	// Insert new permissions
	for _, p := range permissions {
		if p.ID == uuid.Nil {
			p.ID = uuid.New()
		}
		if p.CreatedAt.IsZero() {
			p.CreatedAt = time.Now().UTC()
		}
		_, err := tx.ExecContext(ctx,
			`INSERT INTO rbac_permissions (id, role, resource, action, created_at) VALUES ($1, $2, $3, $4, $5)`,
			p.ID, p.Role, p.Resource, p.Action, p.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert permission: %w", err)
		}
	}

	return tx.Commit()
}
