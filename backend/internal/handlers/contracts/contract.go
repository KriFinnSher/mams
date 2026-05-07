package contracts

//go:generate mockgen -source=contract.go -destination=mocks/contract.go -package=mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/mams/backend/internal/models"
)

type ServiceReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (models.Service, error)
}

type ProtoReader interface {
	ReadProjectProto(ctx context.Context, repositoryURL, ref string) ([]byte, error)
}

