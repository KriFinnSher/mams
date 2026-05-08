package bootstrap

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mams/backend/internal/auth"
)

func SeedDemo(ctx context.Context, pool *pgxpool.Pool) error {
	orgID, err := upsertOrganization(ctx, pool, "Demo Org", "demo")
	if err != nil {
		return err
	}

	ownerID, err := upsertUser(ctx, pool, "owner", "owner", orgID)
	if err != nil {
		return err
	}
	devID, err := upsertUser(ctx, pool, "dev", "dev", orgID)
	if err != nil {
		return err
	}

	serviceID, err := upsertService(ctx, pool, orgID, ownerID)
	if err != nil {
		return err
	}
	if err := upsertServiceAccess(ctx, pool, serviceID, devID, "developer"); err != nil {
		return err
	}
	return nil
}

func upsertOrganization(ctx context.Context, pool *pgxpool.Pool, name, slug string) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO organizations(name, slug)
VALUES ($1, $2)
ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
RETURNING id
`, name, slug).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert organization %s: %w", slug, err)
	}
	return id, nil
}

func upsertUser(ctx context.Context, pool *pgxpool.Pool, login, password string, orgID uuid.UUID) (uuid.UUID, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return uuid.Nil, fmt.Errorf("hash password for %s: %w", login, err)
	}
	var id uuid.UUID
	err = pool.QueryRow(ctx, `
INSERT INTO users(login, password_hash, organization_id)
VALUES ($1, $2, $3)
ON CONFLICT (login)
DO UPDATE SET organization_id = EXCLUDED.organization_id
RETURNING id
`, login, hash, orgID).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert user %s: %w", login, err)
	}
	return id, nil
}

func upsertService(ctx context.Context, pool *pgxpool.Pool, orgID, ownerID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO services (
    organization_id, created_by_user_id, owner_user_id, name, description, type, version,
    test_coverage, minimum_test_coverage_enabled, minimum_test_coverage, pii_sensitive,
    responsible_team_ref, importance, repository_url, default_branch, grafana_dashboard_uid, settings
)
VALUES (
    $1, $2, $3, 'user-service', 'demo user service', 'business', 'v1.0.0',
    80, true, 70, false,
    '@infra-team', 'high', 'https://github.com/example/user-service', 'main', 'demo-user-service',
    '{"minimum_test_coverage_enabled": true, "minimum_test_coverage": 70}'::jsonb
)
ON CONFLICT (organization_id, name)
DO UPDATE SET owner_user_id = EXCLUDED.owner_user_id, updated_at = NOW()
RETURNING id
`, orgID, ownerID, ownerID).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert demo service: %w", err)
	}
	return id, nil
}

func upsertServiceAccess(ctx context.Context, pool *pgxpool.Pool, serviceID, userID uuid.UUID, role string) error {
	_, err := pool.Exec(ctx, `
INSERT INTO service_access(service_id, user_id, role)
VALUES ($1, $2, $3)
ON CONFLICT (service_id, user_id)
DO UPDATE SET role = EXCLUDED.role
`, serviceID, userID, role)
	if err != nil {
		return fmt.Errorf("upsert service access: %w", err)
	}
	return nil
}
