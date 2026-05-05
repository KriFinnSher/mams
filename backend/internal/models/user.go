package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID             uuid.UUID
	Login          string
	PasswordHash   string
	OrganizationID uuid.UUID
	CreatedAt      time.Time
}
