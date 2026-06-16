package rbac

import "github.com/google/uuid"

// ListRolesCommand retrieves all roles with their permission sets.
// Requirements: 9.2
type ListRolesCommand struct{}

// UpdateRolePermissionsCommand replaces the permission set for a role.
// Requirements: 9.3
type UpdateRolePermissionsCommand struct {
	ActorID     uuid.UUID
	ActorName   string
	IPAddress   string
	RoleID      string // role name: student, teacher, admin
	Permissions []PermissionInput
}

// PermissionInput represents a single permission to set.
type PermissionInput struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}
