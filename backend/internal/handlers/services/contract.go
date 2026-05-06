package services

//go:generate mockgen -source=contract.go -destination=mocks/contract.go -package=mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/mams/backend/internal/models"
)

type ServiceReader interface {
	ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]models.Service, error)
}
