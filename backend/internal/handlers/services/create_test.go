package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/mams/backend/internal/handlers/services/mocks"
	authmw "github.com/mams/backend/internal/middleware/auth"
	"github.com/mams/backend/internal/models"
	"go.uber.org/mock/gomock"
)

func TestHandlerCreate(t *testing.T) {
	t.Parallel()

	orgID := uuid.MustParse("300157c6-5599-4077-a3b2-922de677cf85")
	userID := uuid.MustParse("26c613d4-574f-47d3-b6d2-f4e12d20a948")
	serviceID := uuid.MustParse("f4b21e0e-31bd-4d3f-a607-d7af0f0f8f8e")

	validReq := map[string]any{
		"name":                          "user-service",
		"description":                   "desc",
		"type":                          "business",
		"test_coverage":                 80,
		"minimum_test_coverage_enabled": true,
		"minimum_test_coverage":         70,
		"pii_sensitive":                 true,
		"responsible_team_ref":          "@infra-team",
		"importance":                    "high",
		"repository_url":                "https://github.com/org/user-service",
		"default_branch":                "main",
		"grafana_dashboard_uid":         "uid123",
	}

	tests := []struct {
		name       string
		body       any
		setup      func(m *mocks.MockServiceReader) *http.Request
		wantStatus int
		wantErr    string
		wantID     string
	}{
		{
			name: "success",
			body: validReq,
			setup: func(m *mocks.MockServiceReader) *http.Request {
				m.EXPECT().Create(gomock.Any(), gomock.AssignableToTypeOf(models.Service{})).DoAndReturn(
					func(_ context.Context, s models.Service) (models.Service, error) {
						if s.OrganizationID != orgID || s.CreatedByUserID != userID || s.OwnerUserID != userID {
							t.Fatalf("unexpected claims mapping: %+v", s)
						}
						return models.Service{ID: serviceID}, nil
					},
				)
				req := httptest.NewRequest(http.MethodPost, "/api/services", nil)
				return req.WithContext(authmw.WithClaims(req.Context(), authmw.Claims{OrganizationID: orgID, UserID: userID}))
			},
			wantStatus: http.StatusCreated,
			wantID:     serviceID.String(),
		},
		{
			name: "invalid json",
			body: "{bad",
			setup: func(_ *mocks.MockServiceReader) *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/api/services", nil)
				return req.WithContext(authmw.WithClaims(req.Context(), authmw.Claims{OrganizationID: orgID, UserID: userID}))
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid request body",
		},
		{
			name: "validation error",
			body: map[string]any{"name": "", "type": "business"},
			setup: func(_ *mocks.MockServiceReader) *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/api/services", nil)
				return req.WithContext(authmw.WithClaims(req.Context(), authmw.Claims{OrganizationID: orgID, UserID: userID}))
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    "name is required",
		},
		{
			name: "repo error",
			body: validReq,
			setup: func(m *mocks.MockServiceReader) *http.Request {
				m.EXPECT().Create(gomock.Any(), gomock.AssignableToTypeOf(models.Service{})).Return(models.Service{}, errors.New("db error"))
				req := httptest.NewRequest(http.MethodPost, "/api/services", nil)
				return req.WithContext(authmw.WithClaims(req.Context(), authmw.Claims{OrganizationID: orgID, UserID: userID}))
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    "internal error",
		},
		{
			name: "no claims",
			body: validReq,
			setup: func(_ *mocks.MockServiceReader) *http.Request {
				return httptest.NewRequest(http.MethodPost, "/api/services", nil)
			},
			wantStatus: http.StatusUnauthorized,
			wantErr:    "unauthorized",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			reader := mocks.NewMockServiceReader(ctrl)
			h := NewHandler(reader)
			req := tt.setup(reader)
			rec := httptest.NewRecorder()

			var body []byte
			switch v := tt.body.(type) {
			case string:
				body = []byte(v)
			default:
				var err error
				body, err = json.Marshal(v)
				if err != nil {
					t.Fatalf("marshal: %v", err)
				}
			}
			req.Body = io.NopCloser(bytes.NewReader(body))

			h.Create(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantErr != "" {
				var errBody map[string]string
				if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
					t.Fatalf("unmarshal error body: %v", err)
				}
				if errBody["error"] != tt.wantErr {
					t.Fatalf("error = %q, want %q", errBody["error"], tt.wantErr)
				}
				return
			}

			var okBody map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &okBody); err != nil {
				t.Fatalf("unmarshal success body: %v", err)
			}
			if okBody["id"] != tt.wantID {
				t.Fatalf("id = %q, want %q", okBody["id"], tt.wantID)
			}
		})
	}
}
