package rbac

import (
	"context"
	"fmt"
	"time"

	domainrbac "lms-backend/internal/domain/rbac"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// AuditLogger defines the audit logging interface needed by this service.
type AuditLogger interface {
	LogAction(ctx context.Context, actorID uuid.UUID, actorName, action, targetType string, targetID uuid.UUID, metadata map[string]interface{}, ipAddress string) error
}

// Service defines the RBAC use cases.
// Requirements: 9.1–9.4
type Service interface {
	// ListRoles returns all roles with their current permission sets. Requirements: 9.2
	ListRoles(ctx context.Context) (*ListRolesResponse, error)

	// UpdateRolePermissions replaces permissions for a role and records audit log. Requirements: 9.3
	UpdateRolePermissions(ctx context.Context, cmd UpdateRolePermissionsCommand) (*UpdateRolePermissionsResponse, error)
}

// ServiceDeps groups all dependencies for the RBAC service.
type ServiceDeps struct {
	PermissionRepo domainrbac.PermissionRepository
	AuditLogger    AuditLogger
}

type service struct {
	permRepo    domainrbac.PermissionRepository
	auditLogger AuditLogger
}

// NewService creates a new RBAC service.
func NewService(deps ServiceDeps) Service {
	return &service{
		permRepo:    deps.PermissionRepo,
		auditLogger: deps.AuditLogger,
	}
}

// validRoles is the set of built-in roles. Requirements: 9.1
var validRoles = map[string]bool{"student": true, "teacher": true, "admin": true}

// ListRoles returns all roles with their current permission sets.
// Requirements: 9.2
func (s *service) ListRoles(ctx context.Context) (*ListRolesResponse, error) {
	all, err := s.permRepo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}

	roles := make([]RoleResponse, 0, len(all))
	for _, rp := range all {
		roles = append(roles, toRoleResponse(rp))
	}
	return &ListRolesResponse{Data: roles}, nil
}

// UpdateRolePermissions replaces the permission set for a role and records an audit log entry.
// Requirements: 9.3, 9.4
func (s *service) UpdateRolePermissions(ctx context.Context, cmd UpdateRolePermissionsCommand) (*UpdateRolePermissionsResponse, error) {
	if !validRoles[cmd.RoleID] {
		return nil, apperrors.NewNotFoundError("ROLE_NOT_FOUND", "role not found: "+cmd.RoleID)
	}

	perms := make([]domainrbac.Permission, 0, len(cmd.Permissions))
	for _, p := range cmd.Permissions {
		if p.Resource == "" || p.Action == "" {
			return nil, apperrors.NewSimpleValidationError("VALIDATION_ERROR", "each permission must have resource and action")
		}
		perms = append(perms, domainrbac.Permission{
			ID:        uuid.New(),
			Role:      cmd.RoleID,
			Resource:  p.Resource,
			Action:    p.Action,
			CreatedAt: time.Now().UTC(),
		})
	}

	if err := s.permRepo.ReplaceForRole(ctx, cmd.RoleID, perms); err != nil {
		return nil, fmt.Errorf("replace permissions: %w", err)
	}

	// Audit log — action "permission_changed" (Requirement 9.4)
	if s.auditLogger != nil {
		_ = s.auditLogger.LogAction(ctx, cmd.ActorID, cmd.ActorName, "permission_changed", "rbac_role", uuid.Nil,
			map[string]interface{}{"role": cmd.RoleID, "permission_count": len(perms)}, cmd.IPAddress)
	}

	respPerms := make([]PermissionResponse, 0, len(perms))
	for _, p := range perms {
		respPerms = append(respPerms, PermissionResponse{Resource: p.Resource, Action: p.Action})
	}
	return &UpdateRolePermissionsResponse{Role: cmd.RoleID, Permissions: respPerms}, nil
}
