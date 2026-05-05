package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mams/backend/internal/models"
)

var ErrUserNotFound = errors.New("user not found")

type rowScanner interface {
	Scan(dest ...any) error
}

type userQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type UserRepository struct {
	q userQueryer
}

func NewUserRepository(q userQueryer) *UserRepository {
	return &UserRepository{q: q}
}

func (r *UserRepository) Create(ctx context.Context, u models.User) (models.User, error) {
	const q = `
INSERT INTO users (login, password_hash, organization_id)
VALUES ($1, $2, $3)
RETURNING id, login, password_hash, organization_id, created_at
`

	created, err := scanUser(r.q.QueryRow(ctx, q, u.Login, u.PasswordHash, u.OrganizationID))
	if err != nil {
		return models.User{}, err
	}

	return created, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	const q = `
SELECT id, login, password_hash, organization_id, created_at
FROM users
WHERE id = $1
`

	u, err := scanUser(r.q.QueryRow(ctx, q, id))
	if err != nil {
		return models.User{}, mapUserErr(err)
	}

	return u, nil
}

func (r *UserRepository) GetByLogin(ctx context.Context, login string) (models.User, error) {
	const q = `
SELECT id, login, password_hash, organization_id, created_at
FROM users
WHERE login = $1
`

	u, err := scanUser(r.q.QueryRow(ctx, q, login))
	if err != nil {
		return models.User{}, mapUserErr(err)
	}

	return u, nil
}

func scanUser(r rowScanner) (models.User, error) {
	var u models.User
	if err := r.Scan(&u.ID, &u.Login, &u.PasswordHash, &u.OrganizationID, &u.CreatedAt); err != nil {
		return models.User{}, err
	}

	return u, nil
}

func mapUserErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUserNotFound
	}

	return err
}
