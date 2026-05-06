package services

import (
	"bytes"
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
	"github.com/mams/backend/internal/ws"
	"go.uber.org/mock/gomock"
)

func TestHandlerUpdateSettings(t *testing.T) {
	orgID := uuid.New()
	otherOrgID := uuid.New()
	serviceID := uuid.New()

	validBody := map[string]any{
		"settings": map[string]any{
			"minimum_test_coverage_enabled": true,
			"minimum_test_coverage":         70,
		},
	}

	tests := []struct {
		name  string
		body  any
		setup func(m *mocks.MockServiceReader) *http.Request
		want  int
	}{
		{
			name: "ok",
			body: validBody,
			setup: func(m *mocks.MockServiceReader) *http.Request {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{ID: serviceID, OrganizationID: orgID}, nil)
				m.EXPECT().UpdateSettings(gomock.Any(), serviceID, gomock.Any()).Return(models.Service{
					ID: serviceID, Settings: map[string]any{"minimum_test_coverage_enabled": true, "minimum_test_coverage": float64(70)},
				}, nil)
				req := httptest.NewRequest(http.MethodPut, "/api/services/"+serviceID.String()+"/settings", nil)
				req.SetPathValue("id", serviceID.String())
				return req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: orgID}))
			},
			want: http.StatusOK,
		},
		{
			name: "invalid min coverage",
			body: map[string]any{"settings": map[string]any{"minimum_test_coverage_enabled": true, "minimum_test_coverage": 120}},
			setup: func(m *mocks.MockServiceReader) *http.Request {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{ID: serviceID, OrganizationID: orgID}, nil)
				req := httptest.NewRequest(http.MethodPut, "/api/services/"+serviceID.String()+"/settings", nil)
				req.SetPathValue("id", serviceID.String())
				return req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: orgID}))
			},
			want: http.StatusBadRequest,
		},
		{
			name: "service not found",
			body: validBody,
			setup: func(m *mocks.MockServiceReader) *http.Request {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{}, postgresrepo.ErrServiceNotFound)
				req := httptest.NewRequest(http.MethodPut, "/api/services/"+serviceID.String()+"/settings", nil)
				req.SetPathValue("id", serviceID.String())
				return req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: orgID}))
			},
			want: http.StatusNotFound,
		},
		{
			name: "other org",
			body: validBody,
			setup: func(m *mocks.MockServiceReader) *http.Request {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{ID: serviceID, OrganizationID: otherOrgID}, nil)
				req := httptest.NewRequest(http.MethodPut, "/api/services/"+serviceID.String()+"/settings", nil)
				req.SetPathValue("id", serviceID.String())
				return req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: orgID}))
			},
			want: http.StatusNotFound,
		},
		{
			name: "update error",
			body: validBody,
			setup: func(m *mocks.MockServiceReader) *http.Request {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{ID: serviceID, OrganizationID: orgID}, nil)
				m.EXPECT().UpdateSettings(gomock.Any(), serviceID, gomock.Any()).Return(models.Service{}, errors.New("db"))
				req := httptest.NewRequest(http.MethodPut, "/api/services/"+serviceID.String()+"/settings", nil)
				req.SetPathValue("id", serviceID.String())
				return req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: orgID}))
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "owner only forbidden",
			body: validBody,
			setup: func(m *mocks.MockServiceReader) *http.Request {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{ID: serviceID, OrganizationID: orgID, OwnerUserID: uuid.New()}, nil)
				req := httptest.NewRequest(http.MethodPut, "/api/services/"+serviceID.String()+"/settings", nil)
				req.SetPathValue("id", serviceID.String())
				return req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: orgID, UserID: uuid.New()}))
			},
			want: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks.NewMockServiceReader(ctrl)
			h := NewHandler(m, nil, ws.NewHub(), logx.New(slog.New(slog.NewTextHandler(io.Discard, nil))))
			req := tt.setup(m)
			b, _ := json.Marshal(tt.body)
			req.Body = io.NopCloser(bytes.NewReader(b))
			rec := httptest.NewRecorder()
			h.UpdateSettings(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("status=%d want=%d body=%s", rec.Code, tt.want, rec.Body.String())
			}
		})
	}
}

func TestHandlerGetSettings(t *testing.T) {
	orgID := uuid.New()
	otherOrgID := uuid.New()
	serviceID := uuid.New()

	tests := []struct {
		name  string
		setup func(m *mocks.MockServiceReader) *http.Request
		want  int
	}{
		{
			name: "ok",
			setup: func(m *mocks.MockServiceReader) *http.Request {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{
					ID: serviceID, OrganizationID: orgID, Settings: map[string]any{"minimum_test_coverage_enabled": true},
				}, nil)
				req := httptest.NewRequest(http.MethodGet, "/api/services/"+serviceID.String()+"/settings", nil)
				req.SetPathValue("id", serviceID.String())
				return req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: orgID, UserID: uuid.New()}))
			},
			want: http.StatusOK,
		},
		{
			name: "other org",
			setup: func(m *mocks.MockServiceReader) *http.Request {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{ID: serviceID, OrganizationID: otherOrgID}, nil)
				req := httptest.NewRequest(http.MethodGet, "/api/services/"+serviceID.String()+"/settings", nil)
				req.SetPathValue("id", serviceID.String())
				return req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: orgID, UserID: uuid.New()}))
			},
			want: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks.NewMockServiceReader(ctrl)
			h := NewHandler(m, nil, ws.NewHub(), logx.New(slog.New(slog.NewTextHandler(io.Discard, nil))))
			req := tt.setup(m)
			rec := httptest.NewRecorder()
			h.GetSettings(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("status=%d want=%d body=%s", rec.Code, tt.want, rec.Body.String())
			}
		})
	}
}
