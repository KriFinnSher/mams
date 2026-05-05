package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mams/backend/internal/models"
)

var ErrServiceAccessNotFound = errors.New("service access not found")

type ServiceAccessRepository struct {
	q serviceQueryer
}

func NewServiceAccessRepository(q serviceQueryer) *ServiceAccessRepository {
	return &ServiceAccessRepository{q: q}
}

func (r *ServiceAccessRepository) GrantDeveloper(ctx context.Context, serviceID, userID uuid.UUID) (models.ServiceAccess, error) {
	const q = `
INSERT INTO service_access (service_id, user_id, role)
VALUES ($1, $2, 'developer')
ON CONFLICT (service_id, user_id)
DO UPDATE SET role = EXCLUDED.role
RETURNING id, service_id, user_id, role, created_at
`

	a, err := scanServiceAccess(r.q.QueryRow(ctx, q, serviceID, userID))
	if err != nil {
		return models.ServiceAccess{}, err
	}

	return a, nil
}

func (r *ServiceAccessRepository) GetByServiceAndUser(ctx context.Context, serviceID, userID uuid.UUID) (models.ServiceAccess, error) {
	const q = `
SELECT id, service_id, user_id, role, created_at
FROM service_access
WHERE service_id = $1 AND user_id = $2
`

	a, err := scanServiceAccess(r.q.QueryRow(ctx, q, serviceID, userID))
	if err != nil {
		return models.ServiceAccess{}, mapServiceAccessErr(err)
	}

	return a, nil
}

func (r *ServiceAccessRepository) Revoke(ctx context.Context, serviceID, userID uuid.UUID) error {
	const q = `
DELETE FROM service_access
WHERE service_id = $1 AND user_id = $2
RETURNING id
`

	var id uuid.UUID
	if err := r.q.QueryRow(ctx, q, serviceID, userID).Scan(&id); err != nil {
		return mapServiceAccessErr(err)
	}

	return nil
}

func (r *ServiceAccessRepository) ListUserNonObserverAccess(ctx context.Context, userID, orgID uuid.UUID) ([]models.ServiceAccess, error) {
	const q = `
SELECT sa.id, sa.service_id, sa.user_id, sa.role, sa.created_at
FROM service_access sa
JOIN services s ON s.id = sa.service_id
WHERE sa.user_id = $1
  AND s.organization_id = $2
ORDER BY sa.created_at DESC
`

	rows, err := r.q.Query(ctx, q, userID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.ServiceAccess, 0)
	for rows.Next() {
		a, scanErr := scanServiceAccess(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, a)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return out, nil
}

func scanServiceAccess(r rowScanner) (models.ServiceAccess, error) {
	var a models.ServiceAccess
	if err := r.Scan(&a.ID, &a.ServiceID, &a.UserID, &a.Role, &a.CreatedAt); err != nil {
		return models.ServiceAccess{}, err
	}

	return a, nil
}

func mapServiceAccessErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrServiceAccessNotFound
	}

	return err
}
