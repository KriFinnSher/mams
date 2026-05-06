package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/mams/backend/internal/models"
)

type ProfileReader struct {
	users    *UserRepository
	services *ServiceRepository
}

func NewProfileReader(users *UserRepository, services *ServiceRepository) *ProfileReader {
	return &ProfileReader{users: users, services: services}
}

func (r *ProfileReader) GetByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	return r.users.GetByID(ctx, id)
}

func (r *ProfileReader) GetByLogin(ctx context.Context, login string) (models.User, error) {
	return r.users.GetByLogin(ctx, login)
}

func (r *ProfileReader) ListUserNonObserverRoles(ctx context.Context, userID, orgID uuid.UUID) ([]models.ProfileServiceRole, error) {
	return r.services.ListUserNonObserverRoles(ctx, userID, orgID)
}
