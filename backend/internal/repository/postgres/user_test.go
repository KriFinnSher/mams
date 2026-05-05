package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mams/backend/internal/models"
)

type testRow struct {
	user models.User
	err  error
}

func (r testRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}

	*(dest[0].(*uuid.UUID)) = r.user.ID
	*(dest[1].(*string)) = r.user.Login
	*(dest[2].(*string)) = r.user.PasswordHash
	*(dest[3].(*uuid.UUID)) = r.user.OrganizationID
	*(dest[4].(*time.Time)) = r.user.CreatedAt

	return nil
}

type testQueryer struct {
	row rowScanner
}

func (q testQueryer) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	return q.row
}

func TestUserRepositoryCreate(t *testing.T) {
	now := time.Now().UTC()
	orgID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name    string
		row     rowScanner
		in      models.User
		want    models.User
		wantErr error
	}{
		{
			name: "creates user",
			row: testRow{
				user: models.User{
					ID:             userID,
					Login:          "vadim",
					PasswordHash:   "hash",
					OrganizationID: orgID,
					CreatedAt:      now,
				},
			},
			in: models.User{
				Login:          "vadim",
				PasswordHash:   "hash",
				OrganizationID: orgID,
			},
			want: models.User{
				ID:             userID,
				Login:          "vadim",
				PasswordHash:   "hash",
				OrganizationID: orgID,
				CreatedAt:      now,
			},
		},
		{
			name:    "returns db error",
			row:     testRow{err: errors.New("db error")},
			in:      models.User{Login: "vadim"},
			wantErr: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewUserRepository(testQueryer{row: tt.row})
			got, err := repo.Create(context.Background(), tt.in)
			if tt.wantErr != nil {
				if err == nil || err.Error() != tt.wantErr.Error() {
					t.Fatalf("Create() err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Create() err = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Create() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestUserRepositoryGetByID(t *testing.T) {
	now := time.Now().UTC()
	orgID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name    string
		row     rowScanner
		wantErr error
	}{
		{
			name: "returns user",
			row: testRow{
				user: models.User{
					ID:             userID,
					Login:          "vadim",
					PasswordHash:   "hash",
					OrganizationID: orgID,
					CreatedAt:      now,
				},
			},
		},
		{
			name:    "maps not found",
			row:     testRow{err: pgx.ErrNoRows},
			wantErr: ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewUserRepository(testQueryer{row: tt.row})
			_, err := repo.GetByID(context.Background(), userID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("GetByID() err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetByID() err = %v", err)
			}
		})
	}
}

func TestUserRepositoryGetByLogin(t *testing.T) {
	now := time.Now().UTC()
	orgID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name    string
		row     rowScanner
		wantErr error
	}{
		{
			name: "returns user",
			row: testRow{
				user: models.User{
					ID:             userID,
					Login:          "vadim",
					PasswordHash:   "hash",
					OrganizationID: orgID,
					CreatedAt:      now,
				},
			},
		},
		{
			name:    "maps not found",
			row:     testRow{err: pgx.ErrNoRows},
			wantErr: ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewUserRepository(testQueryer{row: tt.row})
			_, err := repo.GetByLogin(context.Background(), "vadim")
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("GetByLogin() err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetByLogin() err = %v", err)
			}
		})
	}
}
