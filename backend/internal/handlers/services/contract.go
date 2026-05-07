package services

//go:generate mockgen -source=contract.go -destination=mocks/contract.go -package=mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/mams/backend/internal/models"
)

type ServiceReader interface {
	Create(ctx context.Context, s models.Service) (models.Service, error)
	GetByID(ctx context.Context, id uuid.UUID) (models.Service, error)
	ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]models.Service, error)
	UpdateInfo(ctx context.Context, s models.Service) (models.Service, error)
	UpdateSettings(ctx context.Context, id uuid.UUID, settings map[string]any) (models.Service, error)
}

type LogReader interface {
	ListByService(ctx context.Context, serviceID uuid.UUID, filter models.LogFilter) ([]models.LogEntry, error)
	Append(ctx context.Context, serviceID uuid.UUID, env, level, message string) *models.LogEntry
}

type ProtoReader interface {
	ReadProjectProto(ctx context.Context, repositoryURL, ref string) ([]byte, error)
}

type ReleaseReader interface {
	ListByService(ctx context.Context, serviceID uuid.UUID) ([]models.Release, error)
}
