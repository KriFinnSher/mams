package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mams/backend/internal/handlers/services/mocks"
	"github.com/mams/backend/internal/logx"
	authmw "github.com/mams/backend/internal/middleware/auth"
	"github.com/mams/backend/internal/models"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
	"github.com/mams/backend/internal/ws"
	"go.uber.org/mock/gomock"
)

type testReleaseReader struct {
	list []models.Release
	err  error
}

func (r testReleaseReader) ListByService(_ context.Context, _ uuid.UUID) ([]models.Release, error) {
	return r.list, r.err
}

func (r testReleaseReader) Create(_ context.Context, rel models.Release) (models.Release, error) {
	if r.err != nil {
		return models.Release{}, r.err
	}
	return rel, nil
}

func TestHandlerGetReleases(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	otherOrgID := uuid.New()
	serviceID := uuid.New()

	tests := []struct {
		name       string
		setupRepo  func(m *mocks.MockServiceReader)
		releases   testReleaseReader
		wantStatus int
		wantErr    string
	}{
		{
			name: "success",
			setupRepo: func(m *mocks.MockServiceReader) {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{
					ID: serviceID, OrganizationID: orgID,
				}, nil)
			},
			releases: testReleaseReader{list: []models.Release{
				{ID: uuid.New(), ServiceID: serviceID, GitTag: "v1.0.0", Branch: "main", Environment: "prod", Strategy: "rolling", Status: "success", Description: "ok", AuthorUserID: uuid.New(), DeployedAt: time.Now()},
			}},
			wantStatus: http.StatusOK,
		},
		{
			name: "invalid id",
			setupRepo: func(_ *mocks.MockServiceReader) {},
			releases: testReleaseReader{},
			wantStatus: http.StatusBadRequest,
			wantErr: "invalid service id",
		},
		{
			name: "service not found",
			setupRepo: func(m *mocks.MockServiceReader) {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{}, postgresrepo.ErrServiceNotFound)
			},
			releases: testReleaseReader{},
			wantStatus: http.StatusNotFound,
			wantErr: "service not found",
		},
		{
			name: "other org",
			setupRepo: func(m *mocks.MockServiceReader) {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{
					ID: serviceID, OrganizationID: otherOrgID,
				}, nil)
			},
			releases: testReleaseReader{},
			wantStatus: http.StatusNotFound,
			wantErr: "service not found",
		},
		{
			name: "releases error",
			setupRepo: func(m *mocks.MockServiceReader) {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{
					ID: serviceID, OrganizationID: orgID,
				}, nil)
			},
			releases: testReleaseReader{err: errors.New("db")},
			wantStatus: http.StatusInternalServerError,
			wantErr: "internal error",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockServiceReader(ctrl)
			tt.setupRepo(repo)
			h := NewHandler(repo, nil, nil, tt.releases, ws.NewHub(), logx.New(slog.New(slog.NewTextHandler(io.Discard, nil))))
			pathID := serviceID.String()
			if tt.name == "invalid id" {
				pathID = "bad-id"
			}
			req := httptest.NewRequest(http.MethodGet, "/api/services/"+pathID+"/releases", nil)
			req.SetPathValue("id", pathID)
			req = req.WithContext(authmw.WithClaims(req.Context(), authmw.Claims{OrganizationID: orgID}))
			rec := httptest.NewRecorder()

			h.GetReleases(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantErr != "" {
				var body map[string]string
				_ = json.Unmarshal(rec.Body.Bytes(), &body)
				if body["error"] != tt.wantErr {
					t.Fatalf("error = %q, want %q", body["error"], tt.wantErr)
				}
			}
		})
	}
}
