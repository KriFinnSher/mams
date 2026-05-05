package models

import (
	"time"

	"github.com/google/uuid"
)

type Release struct {
	ID           uuid.UUID
	ServiceID    uuid.UUID
	GitTag       string
	Branch       string
	Environment  string
	Strategy     string
	Status       string
	Description  string
	AuthorUserID uuid.UUID
	DeployedAt   time.Time
}
