package models

import (
	"time"

	"github.com/google/uuid"
)

type LogEntry struct {
	ID          string
	ServiceID   uuid.UUID
	Environment string
	Level       string
	Message     string
	Timestamp   time.Time
}

type LogFilter struct {
	Level     string
	Text      string
	TimeFrom  *time.Time
	TimeTo    *time.Time
	Limit     int64
}
