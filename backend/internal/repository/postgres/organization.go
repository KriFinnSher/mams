package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrOrganizationNotFound = errors.New("organization not found")

type OrganizationRepository struct {
	q userQueryer
}

func NewOrganizationRepository(q userQueryer) *OrganizationRepository {
	return &OrganizationRepository{q: q}
}

func (r *OrganizationRepository) GetSlugByID(ctx context.Context, id uuid.UUID) (string, error) {
	const q = `
SELECT slug
FROM organizations
WHERE id = $1
`
	var slug string
	if err := r.q.QueryRow(ctx, q, id).Scan(&slug); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrOrganizationNotFound
		}
		return "", err
	}
	return slug, nil
}

