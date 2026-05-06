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
	"go.uber.org/mock/gomock"
)

func TestHandlerUpdateInfo(t *testing.T) {
	orgID := uuid.New()
	otherOrgID := uuid.New()
	serviceID := uuid.New()

	validBody := map[string]any{
		"description":          "new",
		"type":                 "composition",
		"test_coverage":        91,
		"pii_sensitive":        true,
		"responsible_team_ref": "@team",
		"importance":           "critical",
		"repository_url":       "https://github.com/org/repo",
		"default_branch":       "main",
		"grafana_dashboard_uid": "uid2",
	}

	tests := []struct {
		name string
		body any
		setup func(m *mocks.MockServiceReader) *http.Request
		want int
	}{
		{
			name: "ok",
			body: validBody,
			setup: func(m *mocks.MockServiceReader) *http.Request {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{ID: serviceID, OrganizationID: orgID}, nil)
				m.EXPECT().UpdateInfo(gomock.Any(), gomock.AssignableToTypeOf(models.Service{})).Return(models.Service{
					ID: serviceID, Description: "new", Type: "composition", TestCoverage: 91, PIISensitive: true,
					ResponsibleTeamRef: "@team", Importance: "critical", RepositoryURL: "https://github.com/org/repo",
					DefaultBranch: "main", GrafanaDashboardUID: "uid2",
				}, nil)
				req := httptest.NewRequest(http.MethodPut, "/api/services/"+serviceID.String(), nil)
				req.SetPathValue("id", serviceID.String())
				return req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: orgID}))
			},
			want: http.StatusOK,
		},
		{
			name: "not found",
			body: validBody,
			setup: func(m *mocks.MockServiceReader) *http.Request {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{}, postgresrepo.ErrServiceNotFound)
				req := httptest.NewRequest(http.MethodPut, "/api/services/"+serviceID.String(), nil)
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
				req := httptest.NewRequest(http.MethodPut, "/api/services/"+serviceID.String(), nil)
				req.SetPathValue("id", serviceID.String())
				return req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: orgID}))
			},
			want: http.StatusNotFound,
		},
		{
			name: "invalid body",
			body: "{bad",
			setup: func(m *mocks.MockServiceReader) *http.Request {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{ID: serviceID, OrganizationID: orgID}, nil)
				req := httptest.NewRequest(http.MethodPut, "/api/services/"+serviceID.String(), nil)
				req.SetPathValue("id", serviceID.String())
				return req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: orgID}))
			},
			want: http.StatusBadRequest,
		},
		{
			name: "update error",
			body: validBody,
			setup: func(m *mocks.MockServiceReader) *http.Request {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{ID: serviceID, OrganizationID: orgID}, nil)
				m.EXPECT().UpdateInfo(gomock.Any(), gomock.AssignableToTypeOf(models.Service{})).Return(models.Service{}, errors.New("db"))
				req := httptest.NewRequest(http.MethodPut, "/api/services/"+serviceID.String(), nil)
				req.SetPathValue("id", serviceID.String())
				return req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: orgID}))
			},
			want: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks.NewMockServiceReader(ctrl)
			h := NewHandler(m, logx.New(slog.New(slog.NewTextHandler(io.Discard, nil))))
			req := tt.setup(m)
			var b []byte
			switch v := tt.body.(type) {
			case string:
				b = []byte(v)
			default:
				b, _ = json.Marshal(v)
			}
			req.Body = io.NopCloser(bytes.NewReader(b))
			rec := httptest.NewRecorder()
			h.UpdateInfo(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("status=%d want=%d body=%s", rec.Code, tt.want, rec.Body.String())
			}
		})
	}
}

