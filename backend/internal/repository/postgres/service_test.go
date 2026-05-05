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

type serviceTestRow struct {
	service models.Service
	err     error
}

func (r serviceTestRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}

	*(dest[0].(*uuid.UUID)) = r.service.ID
	*(dest[1].(*uuid.UUID)) = r.service.OrganizationID
	*(dest[2].(*uuid.UUID)) = r.service.CreatedByUserID
	*(dest[3].(*uuid.UUID)) = r.service.OwnerUserID
	*(dest[4].(*string)) = r.service.Name
	*(dest[5].(*string)) = r.service.Description
	*(dest[6].(*string)) = r.service.Type
	*(dest[7].(*string)) = r.service.Version
	*(dest[8].(*int)) = r.service.TestCoverage
	*(dest[9].(*bool)) = r.service.MinimumTestCoverageEnabled
	*(dest[10].(*int)) = r.service.MinimumTestCoverage
	*(dest[11].(*bool)) = r.service.PIISensitive
	*(dest[12].(*string)) = r.service.ResponsibleTeamRef
	*(dest[13].(*string)) = r.service.Importance
	*(dest[14].(*string)) = r.service.RepositoryURL
	*(dest[15].(*string)) = r.service.DefaultBranch
	*(dest[16].(*string)) = r.service.GrafanaDashboardUID
	*(dest[17].(*time.Time)) = r.service.CreatedAt
	*(dest[18].(*time.Time)) = r.service.UpdatedAt

	return nil
}

type serviceTestRows struct {
	items []serviceTestRow
	idx   int
	err   error
}

func (r *serviceTestRows) Next() bool {
	return r.idx < len(r.items)
}

func (r *serviceTestRows) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	row := r.items[r.idx]
	r.idx++
	return row.Scan(dest...)
}

func (r *serviceTestRows) Err() error {
	return r.err
}

func (r *serviceTestRows) Close() {}

type serviceTestQueryer struct {
	row  rowScanner
	rows serviceRows
	err  error
}

func (q serviceTestQueryer) QueryRow(_ context.Context, _ string, _ ...any) rowScanner {
	return q.row
}

func (q serviceTestQueryer) Query(_ context.Context, _ string, _ ...any) (serviceRows, error) {
	if q.err != nil {
		return nil, q.err
	}
	return q.rows, nil
}

func testService() models.Service {
	now := time.Now().UTC()
	return models.Service{
		ID:                         uuid.New(),
		OrganizationID:             uuid.New(),
		CreatedByUserID:            uuid.New(),
		OwnerUserID:                uuid.New(),
		Name:                       "user-service",
		Description:                "desc",
		Type:                       "business",
		Version:                    "v1.0.0",
		TestCoverage:               80,
		MinimumTestCoverageEnabled: true,
		MinimumTestCoverage:        70,
		PIISensitive:               true,
		ResponsibleTeamRef:         "@team",
		Importance:                 "high",
		RepositoryURL:              "https://github.com/org/user-service",
		DefaultBranch:              "main",
		GrafanaDashboardUID:        "uid123",
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}
}

func TestServiceRepositoryCreate(t *testing.T) {
	svc := testService()

	tests := []struct {
		name    string
		row     rowScanner
		in      models.Service
		wantErr error
	}{
		{name: "creates service", row: serviceTestRow{service: svc}, in: svc},
		{name: "returns db error", row: serviceTestRow{err: errors.New("db error")}, in: svc, wantErr: errors.New("db error")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewServiceRepository(serviceTestQueryer{row: tt.row})
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
			if got != svc {
				t.Fatalf("Create() got = %+v, want %+v", got, svc)
			}
		})
	}
}

func TestServiceRepositoryGetByID(t *testing.T) {
	svc := testService()

	tests := []struct {
		name    string
		row     rowScanner
		wantErr error
	}{
		{name: "returns service", row: serviceTestRow{service: svc}},
		{name: "maps not found", row: serviceTestRow{err: pgx.ErrNoRows}, wantErr: ErrServiceNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewServiceRepository(serviceTestQueryer{row: tt.row})
			_, err := repo.GetByID(context.Background(), svc.ID)
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

func TestServiceRepositoryListByOrganization(t *testing.T) {
	svc1 := testService()
	svc2 := testService()

	tests := []struct {
		name    string
		rows    serviceRows
		err     error
		wantLen int
		wantErr error
	}{
		{
			name: "returns list",
			rows: &serviceTestRows{
				items: []serviceTestRow{{service: svc1}, {service: svc2}},
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
			repo := NewServiceRepository(serviceTestQueryer{rows: tt.rows, err: tt.err})
			got, err := repo.ListByOrganization(context.Background(), svc1.OrganizationID)
			if tt.wantErr != nil {
				if err == nil || err.Error() != tt.wantErr.Error() {
					t.Fatalf("ListByOrganization() err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ListByOrganization() err = %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("ListByOrganization() len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}
