package models

import (
	"time"

	"github.com/google/uuid"
)

type LogEntry struct {
	ID          string    `json:"id"`
	ServiceID   uuid.UUID `json:"service_id"`
	UserID      string    `json:"user_id"`
	Environment string    `json:"environment"`
	Level       string    `json:"level"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
}

type LogFilter struct {
	Level     string
	Text      string
	TimeFrom  *time.Time
	TimeTo    *time.Time
	Limit     int64
}
