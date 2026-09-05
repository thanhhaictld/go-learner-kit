package repository

import (
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsUniqueEmailViolation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "email unique constraint",
			err: fmt.Errorf("create user: %w", &pgconn.PgError{
				Code:           "23505",
				ConstraintName: "ux_users_email",
			}),
			want: true,
		},
		{
			name: "different unique constraint",
			err: &pgconn.PgError{
				Code:           "23505",
				ConstraintName: "users_pkey",
			},
			want: false,
		},
		{
			name: "different postgres error",
			err: &pgconn.PgError{
				Code:           "23503",
				ConstraintName: "ux_users_email",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUniqueEmailViolation(tt.err); got != tt.want {
				t.Errorf("isUniqueEmailViolation() = %v, want %v", got, tt.want)
			}
		})
	}
}
