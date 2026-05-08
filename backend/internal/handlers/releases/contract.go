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

type OrganizationReader interface {
	GetSlugByID(ctx context.Context, id uuid.UUID) (string, error)
}

type ReleaseReader interface {
	ListByService(ctx context.Context, serviceID uuid.UUID) ([]models.Release, error)
	Create(ctx context.Context, rel models.Release) (models.Release, error)
	GetByID(ctx context.Context, id uuid.UUID) (models.Release, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) (models.Release, error)
	UpdateServiceVersion(ctx context.Context, serviceID uuid.UUID, version string) error
}

type WorkflowDispatcher interface {
	DispatchWorkflow(ctx context.Context, repositoryURL, workflowID, ref string, inputs map[string]string) error
}

type KubeDeployer interface {
	UpgradeRolling(ctx context.Context, namespace, name, container, image string) error
	UpgradeRecreate(ctx context.Context, namespace, name, container, image string) error
	ApplyCanaryPatch(ctx context.Context, namespace, name, canaryName, container, image string, replicas int32) error
	RollbackToTag(ctx context.Context, namespace, name, container, image string) error
}
