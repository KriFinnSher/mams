package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mams/backend/internal/models"
)

var ErrServiceNotFound = errors.New("service not found")

type serviceRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}

type serviceQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) rowScanner
	Query(ctx context.Context, sql string, args ...any) (serviceRows, error)
}

type ServiceRepository struct {
	q serviceQueryer
}

func NewServiceRepository(q serviceQueryer) *ServiceRepository {
	return &ServiceRepository{q: q}
}

type servicePoolAdapter struct {
	pool *pgxpool.Pool
}

func (a servicePoolAdapter) QueryRow(ctx context.Context, sql string, args ...any) rowScanner {
	return a.pool.QueryRow(ctx, sql, args...)
}

func (a servicePoolAdapter) Query(ctx context.Context, sql string, args ...any) (serviceRows, error) {
	rows, err := a.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func NewServiceRepositoryPool(pool *pgxpool.Pool) *ServiceRepository {
	return NewServiceRepository(servicePoolAdapter{pool: pool})
}

func (r *ServiceRepository) Create(ctx context.Context, s models.Service) (models.Service, error) {
	const q = `
INSERT INTO services (
    organization_id, created_by_user_id, owner_user_id, name, description, type, version,
    test_coverage, minimum_test_coverage_enabled, minimum_test_coverage, pii_sensitive,
    responsible_team_ref, importance, repository_url, default_branch, grafana_dashboard_uid
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
RETURNING
    id, organization_id, created_by_user_id, owner_user_id, name, description, type, version,
    test_coverage, minimum_test_coverage_enabled, minimum_test_coverage, pii_sensitive,
    responsible_team_ref, importance, repository_url, default_branch, grafana_dashboard_uid,
    created_at, updated_at
`

	created, err := scanService(r.q.QueryRow(
		ctx,
		q,
		s.OrganizationID,
		s.CreatedByUserID,
		s.OwnerUserID,
		s.Name,
		s.Description,
		s.Type,
		s.Version,
		s.TestCoverage,
		s.MinimumTestCoverageEnabled,
		s.MinimumTestCoverage,
		s.PIISensitive,
		s.ResponsibleTeamRef,
		s.Importance,
		s.RepositoryURL,
		s.DefaultBranch,
		s.GrafanaDashboardUID,
	))
	if err != nil {
		return models.Service{}, err
	}

	return created, nil
}

func (r *ServiceRepository) GetByID(ctx context.Context, id uuid.UUID) (models.Service, error) {
	const q = `
SELECT
    id, organization_id, created_by_user_id, owner_user_id, name, description, type, version,
    test_coverage, minimum_test_coverage_enabled, minimum_test_coverage, pii_sensitive,
    responsible_team_ref, importance, repository_url, default_branch, grafana_dashboard_uid,
    created_at, updated_at
FROM services
WHERE id = $1
`

	s, err := scanService(r.q.QueryRow(ctx, q, id))
	if err != nil {
		return models.Service{}, mapServiceErr(err)
	}

	return s, nil
}

func (r *ServiceRepository) ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]models.Service, error) {
	const q = `
SELECT
    id, organization_id, created_by_user_id, owner_user_id, name, description, type, version,
    test_coverage, minimum_test_coverage_enabled, minimum_test_coverage, pii_sensitive,
    responsible_team_ref, importance, repository_url, default_branch, grafana_dashboard_uid,
    created_at, updated_at
FROM services
WHERE organization_id = $1
ORDER BY created_at DESC
`

	rows, err := r.q.Query(ctx, q, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.Service, 0)
	for rows.Next() {
		s, scanErr := scanService(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, s)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return out, nil
}

func (r *ServiceRepository) UpdateInfo(ctx context.Context, s models.Service) (models.Service, error) {
	const q = `
UPDATE services
SET
    description = $2,
    type = $3,
    test_coverage = $4,
    pii_sensitive = $5,
    responsible_team_ref = $6,
    importance = $7,
    repository_url = $8,
    default_branch = $9,
    grafana_dashboard_uid = $10,
    updated_at = NOW()
WHERE id = $1
RETURNING
    id, organization_id, created_by_user_id, owner_user_id, name, description, type, version,
    test_coverage, minimum_test_coverage_enabled, minimum_test_coverage, pii_sensitive,
    responsible_team_ref, importance, repository_url, default_branch, grafana_dashboard_uid,
    created_at, updated_at
`

	updated, err := scanService(r.q.QueryRow(
		ctx,
		q,
		s.ID,
		s.Description,
		s.Type,
		s.TestCoverage,
		s.PIISensitive,
		s.ResponsibleTeamRef,
		s.Importance,
		s.RepositoryURL,
		s.DefaultBranch,
		s.GrafanaDashboardUID,
	))
	if err != nil {
		return models.Service{}, mapServiceErr(err)
	}

	return updated, nil
}

func (r *ServiceRepository) UpdateSettings(ctx context.Context, id uuid.UUID, enabled bool, minimum int) (models.Service, error) {
	const q = `
UPDATE services
SET
    minimum_test_coverage_enabled = $2,
    minimum_test_coverage = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING
    id, organization_id, created_by_user_id, owner_user_id, name, description, type, version,
    test_coverage, minimum_test_coverage_enabled, minimum_test_coverage, pii_sensitive,
    responsible_team_ref, importance, repository_url, default_branch, grafana_dashboard_uid,
    created_at, updated_at
`

	updated, err := scanService(r.q.QueryRow(ctx, q, id, enabled, minimum))
	if err != nil {
		return models.Service{}, mapServiceErr(err)
	}

	return updated, nil
}

func scanService(r rowScanner) (models.Service, error) {
	var s models.Service
	if err := r.Scan(
		&s.ID,
		&s.OrganizationID,
		&s.CreatedByUserID,
		&s.OwnerUserID,
		&s.Name,
		&s.Description,
		&s.Type,
		&s.Version,
		&s.TestCoverage,
		&s.MinimumTestCoverageEnabled,
		&s.MinimumTestCoverage,
		&s.PIISensitive,
		&s.ResponsibleTeamRef,
		&s.Importance,
		&s.RepositoryURL,
		&s.DefaultBranch,
		&s.GrafanaDashboardUID,
		&s.CreatedAt,
		&s.UpdatedAt,
	); err != nil {
		return models.Service{}, err
	}

	return s, nil
}

func mapServiceErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrServiceNotFound
	}

	return err
}
