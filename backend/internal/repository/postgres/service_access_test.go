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

type serviceAccessTestRow struct {
	access models.ServiceAccess
	id     uuid.UUID
	err    error
}

func (r serviceAccessTestRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) == 1 {
		*(dest[0].(*uuid.UUID)) = r.id
		return nil
	}

	*(dest[0].(*uuid.UUID)) = r.access.ID
	*(dest[1].(*uuid.UUID)) = r.access.ServiceID
	*(dest[2].(*uuid.UUID)) = r.access.UserID
	*(dest[3].(*string)) = r.access.Role
	*(dest[4].(*time.Time)) = r.access.CreatedAt
	return nil
}

func testServiceAccess() models.ServiceAccess {
	return models.ServiceAccess{
		ID:        uuid.New(),
		ServiceID: uuid.New(),
		UserID:    uuid.New(),
		Role:      "developer",
		CreatedAt: time.Now().UTC(),
	}
}

func TestServiceAccessRepositoryGrantDeveloper(t *testing.T) {
	a := testServiceAccess()

	tests := []struct {
		name    string
		row     rowScanner
		wantErr error
	}{
		{name: "grants access", row: serviceAccessTestRow{access: a}},
		{name: "returns db error", row: serviceAccessTestRow{err: errors.New("db error")}, wantErr: errors.New("db error")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewServiceAccessRepository(serviceTestQueryer{row: tt.row})
			got, err := repo.GrantDeveloper(context.Background(), a.ServiceID, a.UserID)
			if tt.wantErr != nil {
				if err == nil || err.Error() != tt.wantErr.Error() {
					t.Fatalf("GrantDeveloper() err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GrantDeveloper() err = %v", err)
			}
			if got != a {
				t.Fatalf("GrantDeveloper() got = %+v, want %+v", got, a)
			}
		})
	}
}

func TestServiceAccessRepositoryGetByServiceAndUser(t *testing.T) {
	a := testServiceAccess()

	tests := []struct {
		name    string
		row     rowScanner
		wantErr error
	}{
		{name: "returns access", row: serviceAccessTestRow{access: a}},
		{name: "maps not found", row: serviceAccessTestRow{err: pgx.ErrNoRows}, wantErr: ErrServiceAccessNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewServiceAccessRepository(serviceTestQueryer{row: tt.row})
			_, err := repo.GetByServiceAndUser(context.Background(), a.ServiceID, a.UserID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("GetByServiceAndUser() err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetByServiceAndUser() err = %v", err)
			}
		})
	}
}

func TestServiceAccessRepositoryRevoke(t *testing.T) {
	a := testServiceAccess()

	tests := []struct {
		name    string
		row     rowScanner
		wantErr error
	}{
		{name: "revokes access", row: serviceAccessTestRow{id: uuid.New()}},
		{name: "maps not found", row: serviceAccessTestRow{err: pgx.ErrNoRows}, wantErr: ErrServiceAccessNotFound},
		{name: "returns db error", row: serviceAccessTestRow{err: errors.New("db error")}, wantErr: errors.New("db error")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewServiceAccessRepository(serviceTestQueryer{row: tt.row})
			err := repo.Revoke(context.Background(), a.ServiceID, a.UserID)
			if tt.wantErr != nil {
				if errors.Is(tt.wantErr, ErrServiceAccessNotFound) {
					if !errors.Is(err, tt.wantErr) {
						t.Fatalf("Revoke() err = %v, want %v", err, tt.wantErr)
					}
					return
				}
				if err == nil || err.Error() != tt.wantErr.Error() {
					t.Fatalf("Revoke() err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Revoke() err = %v", err)
			}
		})
	}
}

func TestServiceAccessRepositoryListUserNonObserverAccess(t *testing.T) {
	a1 := testServiceAccess()
	a2 := testServiceAccess()

	tests := []struct {
		name    string
		rows    serviceRows
		err     error
		wantLen int
		wantErr error
	}{
		{
			name:    "returns query error",
			err:     errors.New("query error"),
			wantErr: errors.New("query error"),
		},
		{
			name: "returns access rows",
			rows: &serviceAccessRows{
				items: []serviceAccessTestRow{{access: a1}, {access: a2}},
			},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewServiceAccessRepository(serviceTestQueryer{rows: tt.rows, err: tt.err})
			got, err := repo.ListUserNonObserverAccess(context.Background(), uuid.New(), uuid.New())
			if tt.wantErr != nil {
				if err == nil || err.Error() != tt.wantErr.Error() {
					t.Fatalf("ListUserNonObserverAccess() err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ListUserNonObserverAccess() err = %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("ListUserNonObserverAccess() len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

type serviceAccessRows struct {
	items []serviceAccessTestRow
	idx   int
	err   error
}

func (r *serviceAccessRows) Next() bool {
	return r.idx < len(r.items)
}

func (r *serviceAccessRows) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	row := r.items[r.idx]
	r.idx++
	return row.Scan(dest...)
}

func (r *serviceAccessRows) Err() error {
	return r.err
}

func (r *serviceAccessRows) Close() {}
