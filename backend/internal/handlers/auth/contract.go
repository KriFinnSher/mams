package auth

//go:generate mockgen -source=contract.go -destination=mocks/contract.go -package=mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/mams/backend/internal/models"
)

type UserReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (models.User, error)
	GetByLogin(ctx context.Context, login string) (models.User, error)
}

type TokenIssuer interface {
	IssueToken(user models.User) (string, error)
}
