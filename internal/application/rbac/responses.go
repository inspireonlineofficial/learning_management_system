package rbac

import "lms-backend/internal/domain/rbac"

// RoleResponse is the public representation of a role with its permissions.
// Requirements: 9.2
type RoleResponse struct {
	Role        string               `json:"role"`
	Permissions []PermissionResponse `json:"permissions"`
}

// PermissionResponse is the public representation of a single permission.
type PermissionResponse struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// ListRolesResponse wraps all roles.
type ListRolesResponse struct {
	Data []RoleResponse `json:"data"`
}

// UpdateRolePermissionsResponse confirms the update.
type UpdateRolePermissionsResponse struct {
	Role        string               `json:"role"`
	Permissions []PermissionResponse `json:"permissions"`
}

func toRoleResponse(rp rbac.RolePermissions) RoleResponse {
	perms := make([]PermissionResponse, 0, len(rp.Permissions))
	for _, p := range rp.Permissions {
		perms = append(perms, PermissionResponse{Resource: p.Resource, Action: p.Action})
	}
	return RoleResponse{Role: rp.Role, Permissions: perms}
}
