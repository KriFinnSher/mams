package models

import (
	"time"

	"github.com/google/uuid"
)

type ServiceAccess struct {
	ID        uuid.UUID
	ServiceID uuid.UUID
	UserID    uuid.UUID
	Role      string
	CreatedAt time.Time
}
