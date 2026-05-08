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

type K8sLogReader interface {
	GetPodLogs(ctx context.Context, namespace, labelSelector string, limit int64) (string, error)
}

type ServiceGetter interface {
	GetByID(ctx context.Context, id uuid.UUID) (models.Service, error)
}

type OrgGetter interface {
	GetSlugByID(ctx context.Context, id uuid.UUID) (string, error)
}

