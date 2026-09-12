package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	Id             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organizationId"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	CreatedAt      time.Time `json:"createdAt"`
}
