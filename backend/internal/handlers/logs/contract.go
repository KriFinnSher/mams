package logs

//go:generate mockgen -source=contract.go -destination=mocks/contract.go -package=mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/mams/backend/internal/models"
)

type Reader interface {
	ListByService(ctx context.Context, serviceID uuid.UUID, filter models.LogFilter) ([]models.LogEntry, error)
	Append(ctx context.Context, serviceID uuid.UUID, env, level, message string) *models.LogEntry
}

