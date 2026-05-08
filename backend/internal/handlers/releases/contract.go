package releases

//go:generate mockgen -source=contract.go -destination=mocks/contract.go -package=mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/mams/backend/internal/models"
)

type ServiceReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (models.Service, error)
}

type ReleaseReader interface {
	ListByService(ctx context.Context, serviceID uuid.UUID) ([]models.Release, error)
	Create(ctx context.Context, rel models.Release) (models.Release, error)
	GetByID(ctx context.Context, id uuid.UUID) (models.Release, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) (models.Release, error)
}

type WorkflowDispatcher interface {
	DispatchWorkflow(ctx context.Context, repositoryURL, workflowID, ref string, inputs map[string]string) error
}
