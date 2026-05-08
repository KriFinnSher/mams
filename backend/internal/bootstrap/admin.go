package bootstrap

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mams/backend/internal/auth"
)

const (
	adminLogin    = "admin"
	adminPassword = "admin"
	orgName       = "MAMS"
	orgSlug       = "mams"
)

func SeedAdmin(ctx context.Context, pool *pgxpool.Pool) error {
	var orgID uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO organizations(name, slug)
VALUES ($1, $2)
ON CONFLICT (slug) DO UPDATE SET slug = EXCLUDED.slug
RETURNING id
`, orgName, orgSlug).Scan(&orgID)
	if err != nil {
		return fmt.Errorf("upsert organization: %w", err)
	}

	hash, err := auth.HashPassword(adminPassword)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	if _, err := pool.Exec(ctx, `
INSERT INTO users(login, password_hash, organization_id)
VALUES ($1, $2, $3)
ON CONFLICT (login) DO NOTHING
`, adminLogin, hash, orgID); err != nil {
		return fmt.Errorf("insert admin user: %w", err)
	}

	return nil
}

func SeedAll(ctx context.Context, pool *pgxpool.Pool) error {
	if err := SeedAdmin(ctx, pool); err != nil {
		return err
	}
	if err := SeedDemo(ctx, pool); err != nil {
		return err
	}
	return nil
}
