package authz

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type Permission string

const (
	PermissionListUsers   Permission = "list_users"
	PermissionCreateUsers Permission = "create_user"
	PermissionManageRoles Permission = "manage_roles"
)

var (
	ErrDenied      = errors.New("permission denied")
	ErrUnavailable = errors.New("authorization engine unavailable")
)

func (p Permission) Valid() bool {
	return p == PermissionListUsers || p == PermissionCreateUsers || p == PermissionManageRoles
}

func (p Permission) AssignableToCustomRole() bool {
	return p == PermissionListUsers || p == PermissionCreateUsers
}

type Engine interface {
	Initialize(ctx context.Context) error
	Check(ctx context.Context, subjectID, organizationID uuid.UUID, permission Permission) (bool, error)
	AssignAdmin(ctx context.Context, subjectID, organizationID uuid.UUID) error
	RevokeAdmin(ctx context.Context, subjectID, organizationID uuid.UUID) error
	CreateRole(ctx context.Context, organizationID, roleID uuid.UUID) error
	RoleExists(ctx context.Context, organizationID, roleID uuid.UUID) (bool, error)
	AssignRole(ctx context.Context, subjectID, roleID uuid.UUID) error
	RevokeRole(ctx context.Context, subjectID, roleID uuid.UUID) error
	GrantRolePermission(ctx context.Context, organizationID, roleID uuid.UUID, permission Permission) error
	RevokeRolePermission(ctx context.Context, organizationID, roleID uuid.UUID, permission Permission) error
}
