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
	UpdateSettings(ctx context.Context, id uuid.UUID, enabled bool, minimum int) (models.Service, error)
}
