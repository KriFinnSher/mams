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

type releaseTestRow struct {
	release models.Release
	id      uuid.UUID
	err     error
}

func (r releaseTestRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) == 1 {
		*(dest[0].(*uuid.UUID)) = r.id
		return nil
	}

	*(dest[0].(*uuid.UUID)) = r.release.ID
	*(dest[1].(*uuid.UUID)) = r.release.ServiceID
	*(dest[2].(*string)) = r.release.GitTag
	*(dest[3].(*string)) = r.release.Branch
	*(dest[4].(*string)) = r.release.Environment
	*(dest[5].(*string)) = r.release.Strategy
	*(dest[6].(*string)) = r.release.Status
	*(dest[7].(*string)) = r.release.Description
	*(dest[8].(*uuid.UUID)) = r.release.AuthorUserID
	*(dest[9].(*time.Time)) = r.release.DeployedAt
	return nil
}

type releaseRows struct {
	items []releaseTestRow
	idx   int
	err   error
}

func (r *releaseRows) Next() bool {
	return r.idx < len(r.items)
}

func (r *releaseRows) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	row := r.items[r.idx]
	r.idx++
	return row.Scan(dest...)
}

func (r *releaseRows) Err() error {
	return r.err
}

func (r *releaseRows) Close() {}

func testRelease() models.Release {
	return models.Release{
		ID:           uuid.New(),
		ServiceID:    uuid.New(),
		GitTag:       "v1.0.1",
		Branch:       "main",
		Environment:  "prod",
		Strategy:     "rolling",
		Status:       "pending",
		Description:  "release",
		AuthorUserID: uuid.New(),
		DeployedAt:   time.Now().UTC(),
	}
}

func TestReleaseRepositoryCreate(t *testing.T) {
	rel := testRelease()

	tests := []struct {
		name    string
		row     rowScanner
		wantErr error
	}{
		{name: "creates release", row: releaseTestRow{release: rel}},
		{name: "returns db error", row: releaseTestRow{err: errors.New("db error")}, wantErr: errors.New("db error")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewReleaseRepository(serviceTestQueryer{row: tt.row})
			got, err := repo.Create(context.Background(), rel)
			if tt.wantErr != nil {
				if err == nil || err.Error() != tt.wantErr.Error() {
					t.Fatalf("Create() err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Create() err = %v", err)
			}
			if got != rel {
				t.Fatalf("Create() got = %+v, want %+v", got, rel)
			}
		})
	}
}

func TestReleaseRepositoryGetByID(t *testing.T) {
	rel := testRelease()

	tests := []struct {
		name    string
		row     rowScanner
		wantErr error
	}{
		{name: "returns release", row: releaseTestRow{release: rel}},
		{name: "maps not found", row: releaseTestRow{err: pgx.ErrNoRows}, wantErr: ErrReleaseNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewReleaseRepository(serviceTestQueryer{row: tt.row})
			_, err := repo.GetByID(context.Background(), rel.ID)
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

func TestReleaseRepositoryListByService(t *testing.T) {
	r1 := testRelease()
	r2 := testRelease()

	tests := []struct {
		name    string
		rows    serviceRows
		err     error
		wantLen int
		wantErr error
	}{
		{
			name: "returns release list",
			rows: &releaseRows{
				items: []releaseTestRow{{release: r1}, {release: r2}},
			},
			wantLen: 2,
		},
		{
			name:    "returns query error",
			err:     errors.New("query error"),
			wantErr: errors.New("query error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewReleaseRepository(serviceTestQueryer{rows: tt.rows, err: tt.err})
			got, err := repo.ListByService(context.Background(), r1.ServiceID)
			if tt.wantErr != nil {
				if err == nil || err.Error() != tt.wantErr.Error() {
					t.Fatalf("ListByService() err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ListByService() err = %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("ListByService() len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestReleaseRepositoryUpdateStatus(t *testing.T) {
	rel := testRelease()

	tests := []struct {
		name    string
		row     rowScanner
		wantErr error
	}{
		{name: "updates status", row: releaseTestRow{release: rel}},
		{name: "maps not found", row: releaseTestRow{err: pgx.ErrNoRows}, wantErr: ErrReleaseNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewReleaseRepository(serviceTestQueryer{row: tt.row})
			_, err := repo.UpdateStatus(context.Background(), rel.ID, "success")
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("UpdateStatus() err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdateStatus() err = %v", err)
			}
		})
	}
}

func TestReleaseRepositoryUpdateServiceVersion(t *testing.T) {
	serviceID := uuid.New()

	tests := []struct {
		name    string
		row     rowScanner
		wantErr error
	}{
		{name: "updates service version", row: releaseTestRow{id: serviceID}},
		{name: "maps service not found", row: releaseTestRow{err: pgx.ErrNoRows}, wantErr: ErrServiceNotFound},
		{name: "returns db error", row: releaseTestRow{err: errors.New("db error")}, wantErr: errors.New("db error")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewReleaseRepository(serviceTestQueryer{row: tt.row})
			err := repo.UpdateServiceVersion(context.Background(), serviceID, "v1.0.2")
			if tt.wantErr != nil {
				if errors.Is(tt.wantErr, ErrServiceNotFound) {
					if !errors.Is(err, tt.wantErr) {
						t.Fatalf("UpdateServiceVersion() err = %v, want %v", err, tt.wantErr)
					}
					return
				}
				if err == nil || err.Error() != tt.wantErr.Error() {
					t.Fatalf("UpdateServiceVersion() err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdateServiceVersion() err = %v", err)
			}
		})
	}
}
