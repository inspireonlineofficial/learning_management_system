package rbac

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Permission represents a single role-resource-action permission entry.
// Requirements: 9.1
type Permission struct {
	ID        uuid.UUID `json:"id"`
	Role      string    `json:"role"`     // student, teacher, admin
	Resource  string    `json:"resource"` // courses, enrollments, etc.
	Action    string    `json:"action"`   // read, create, update_own, etc.
	CreatedAt time.Time `json:"created_at"`
}

// RolePermissions groups all permissions for a single role.
// Requirements: 9.2
type RolePermissions struct {
	Role        string       `json:"role"`
	Permissions []Permission `json:"permissions"`
}

// PermissionRepository defines the port for RBAC persistence.
type PermissionRepository interface {
	// ListByRole returns all permissions for a given role.
	ListByRole(ctx context.Context, role string) ([]Permission, error)

	// ListAll returns all permissions grouped by role.
	ListAll(ctx context.Context) ([]RolePermissions, error)

	// ReplaceForRole atomically replaces all permissions for a role.
	// Used by UpdateRolePermissions. Requirements: 9.3
	ReplaceForRole(ctx context.Context, role string, permissions []Permission) error
}
