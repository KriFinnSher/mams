package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mams/backend/internal/models"
)

var ErrReleaseNotFound = errors.New("release not found")

type ReleaseRepository struct {
	q serviceQueryer
}

func NewReleaseRepository(q serviceQueryer) *ReleaseRepository {
	return &ReleaseRepository{q: q}
}

type releasePoolAdapter struct {
	pool *pgxpool.Pool
}

func (a releasePoolAdapter) QueryRow(ctx context.Context, sql string, args ...any) rowScanner {
	return a.pool.QueryRow(ctx, sql, args...)
}

func (a releasePoolAdapter) Query(ctx context.Context, sql string, args ...any) (serviceRows, error) {
	rows, err := a.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func NewReleaseRepositoryPool(pool *pgxpool.Pool) *ReleaseRepository {
	return NewReleaseRepository(releasePoolAdapter{pool: pool})
}

func (r *ReleaseRepository) Create(ctx context.Context, rel models.Release) (models.Release, error) {
	const q = `
INSERT INTO releases (
    service_id, git_tag, branch, environment, strategy, status, description, author_user_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, service_id, git_tag, branch, environment, strategy, status, description, author_user_id, deployed_at
`

	out, err := scanRelease(r.q.QueryRow(
		ctx,
		q,
		rel.ServiceID,
		rel.GitTag,
		rel.Branch,
		rel.Environment,
		rel.Strategy,
		rel.Status,
		rel.Description,
		rel.AuthorUserID,
	))
	if err != nil {
		return models.Release{}, err
	}

	return out, nil
}

func (r *ReleaseRepository) GetByID(ctx context.Context, id uuid.UUID) (models.Release, error) {
	const q = `
SELECT id, service_id, git_tag, branch, environment, strategy, status, description, author_user_id, deployed_at
FROM releases
WHERE id = $1
`

	out, err := scanRelease(r.q.QueryRow(ctx, q, id))
	if err != nil {
		return models.Release{}, mapReleaseErr(err)
	}

	return out, nil
}

func (r *ReleaseRepository) ListByService(ctx context.Context, serviceID uuid.UUID) ([]models.Release, error) {
	const q = `
SELECT id, service_id, git_tag, branch, environment, strategy, status, description, author_user_id, deployed_at
FROM releases
WHERE service_id = $1
ORDER BY deployed_at DESC
`

	rows, err := r.q.Query(ctx, q, serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.Release, 0)
	for rows.Next() {
		rel, scanErr := scanRelease(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, rel)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return out, nil
}

func (r *ReleaseRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (models.Release, error) {
	const q = `
UPDATE releases
SET status = $2
WHERE id = $1
RETURNING id, service_id, git_tag, branch, environment, strategy, status, description, author_user_id, deployed_at
`

	out, err := scanRelease(r.q.QueryRow(ctx, q, id, status))
	if err != nil {
		return models.Release{}, mapReleaseErr(err)
	}

	return out, nil
}

func (r *ReleaseRepository) UpdateServiceVersion(ctx context.Context, serviceID uuid.UUID, version string) error {
	const q = `
UPDATE services
SET version = $2, updated_at = NOW()
WHERE id = $1
RETURNING id
`

	var id uuid.UUID
	if err := r.q.QueryRow(ctx, q, serviceID, version).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrServiceNotFound
		}
		return err
	}

	return nil
}

func scanRelease(r rowScanner) (models.Release, error) {
	var rel models.Release
	if err := r.Scan(
		&rel.ID,
		&rel.ServiceID,
		&rel.GitTag,
		&rel.Branch,
		&rel.Environment,
		&rel.Strategy,
		&rel.Status,
		&rel.Description,
		&rel.AuthorUserID,
		&rel.DeployedAt,
	); err != nil {
		return models.Release{}, err
	}

	return rel, nil
}

func mapReleaseErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrReleaseNotFound
	}

	return err
}
