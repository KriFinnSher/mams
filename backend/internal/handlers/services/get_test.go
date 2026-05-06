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

	"github.com/google/uuid"
	"github.com/mams/backend/internal/handlers/services/mocks"
	"github.com/mams/backend/internal/logx"
	authmw "github.com/mams/backend/internal/middleware/auth"
	"github.com/mams/backend/internal/models"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
	"go.uber.org/mock/gomock"
)

func TestHandlerGet(t *testing.T) {
	t.Parallel()

	orgID := uuid.MustParse("300157c6-5599-4077-a3b2-922de677cf85")
	otherOrgID := uuid.MustParse("bf35790a-93b5-4ca1-9426-c5c37299582e")
	serviceID := uuid.MustParse("f4b21e0e-31bd-4d3f-a607-d7af0f0f8f8e")

	tests := []struct {
		name       string
		pathID     string
		setup      func(m *mocks.MockServiceReader) *http.Request
		wantStatus int
		wantErr    string
	}{
		{
			name:   "success",
			pathID: serviceID.String(),
			setup: func(m *mocks.MockServiceReader) *http.Request {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{
					ID:             serviceID,
					OrganizationID: orgID,
					Name:           "svc",
				}, nil)
				req := httptest.NewRequest(http.MethodGet, "/api/services/"+serviceID.String(), nil)
				req.SetPathValue("id", serviceID.String())
				return req.WithContext(authmw.WithClaims(req.Context(), authmw.Claims{OrganizationID: orgID}))
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "invalid id",
			pathID: "bad-id",
			setup: func(_ *mocks.MockServiceReader) *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/api/services/bad-id", nil)
				req.SetPathValue("id", "bad-id")
				return req.WithContext(authmw.WithClaims(req.Context(), authmw.Claims{OrganizationID: orgID}))
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid service id",
		},
		{
			name:   "not found",
			pathID: serviceID.String(),
			setup: func(m *mocks.MockServiceReader) *http.Request {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{}, postgresrepo.ErrServiceNotFound)
				req := httptest.NewRequest(http.MethodGet, "/api/services/"+serviceID.String(), nil)
				req.SetPathValue("id", serviceID.String())
				return req.WithContext(authmw.WithClaims(req.Context(), authmw.Claims{OrganizationID: orgID}))
			},
			wantStatus: http.StatusNotFound,
			wantErr:    "service not found",
		},
		{
			name:   "other org",
			pathID: serviceID.String(),
			setup: func(m *mocks.MockServiceReader) *http.Request {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{
					ID:             serviceID,
					OrganizationID: otherOrgID,
				}, nil)
				req := httptest.NewRequest(http.MethodGet, "/api/services/"+serviceID.String(), nil)
				req.SetPathValue("id", serviceID.String())
				return req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: orgID}))
			},
			wantStatus: http.StatusNotFound,
			wantErr:    "service not found",
		},
		{
			name:   "repo error",
			pathID: serviceID.String(),
			setup: func(m *mocks.MockServiceReader) *http.Request {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{}, errors.New("db"))
				req := httptest.NewRequest(http.MethodGet, "/api/services/"+serviceID.String(), nil)
				req.SetPathValue("id", serviceID.String())
				return req.WithContext(authmw.WithClaims(req.Context(), authmw.Claims{OrganizationID: orgID}))
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    "internal error",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockServiceReader(ctrl)
			h := NewHandler(repo, nil, logx.New(slog.New(slog.NewTextHandler(io.Discard, nil))))
			req := tt.setup(repo)
			rec := httptest.NewRecorder()

			h.Get(rec, req)

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
