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
)

var (
	ErrDenied      = errors.New("permission denied")
	ErrUnavailable = errors.New("authorization engine unavailable")
)

func (p Permission) Valid() bool {
	return p == PermissionListUsers || p == PermissionCreateUsers
}

type Engine interface {
	Initialize(ctx context.Context) error
	Check(ctx context.Context, subjectID, organizationID uuid.UUID, permission Permission) (bool, error)
	AssignAdmin(ctx context.Context, subjectID, organizationID uuid.UUID) error
	RevokeAdmin(ctx context.Context, subjectID, organizationID uuid.UUID) error
}
