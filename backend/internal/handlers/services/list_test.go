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
	"github.com/mams/backend/internal/ws"
	"go.uber.org/mock/gomock"
)

func TestHandlerList(t *testing.T) {
	t.Parallel()

	orgID := uuid.MustParse("300157c6-5599-4077-a3b2-922de677cf85")
	serviceID := uuid.MustParse("f4b21e0e-31bd-4d3f-a607-d7af0f0f8f8e")

	tests := []struct {
		name       string
		setup      func(m *mocks.MockServiceReader) *http.Request
		wantStatus int
		wantErr    string
		wantName   string
	}{
		{
			name: "success",
			setup: func(m *mocks.MockServiceReader) *http.Request {
				m.EXPECT().ListByOrganization(gomock.Any(), orgID).Return([]models.Service{
					{ID: serviceID, Name: "user-service"},
				}, nil)
				req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
				return req.WithContext(authmw.WithClaims(req.Context(), authmw.Claims{OrganizationID: orgID}))
			},
			wantStatus: http.StatusOK,
			wantName:   "user-service",
		},
		{
			name: "no claims",
			setup: func(_ *mocks.MockServiceReader) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/api/services", nil)
			},
			wantStatus: http.StatusUnauthorized,
			wantErr:    "unauthorized",
		},
		{
			name: "repo error",
			setup: func(m *mocks.MockServiceReader) *http.Request {
				m.EXPECT().ListByOrganization(gomock.Any(), orgID).Return(nil, errors.New("db error"))
				req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
				return req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: orgID}))
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
			reader := mocks.NewMockServiceReader(ctrl)
			h := NewHandler(reader, nil, ws.NewHub(), logx.New(slog.New(slog.NewTextHandler(io.Discard, nil))))
			req := tt.setup(reader)
			rec := httptest.NewRecorder()

			h.List(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantErr != "" {
				var body map[string]string
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("unmarshal error body: %v", err)
				}
				if body["error"] != tt.wantErr {
					t.Fatalf("error = %q, want %q", body["error"], tt.wantErr)
				}
				return
			}

			var body struct {
				Services []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"services"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("unmarshal success body: %v", err)
			}
			if len(body.Services) != 1 {
				t.Fatalf("services len = %d, want 1", len(body.Services))
			}
			if body.Services[0].ID != serviceID.String() {
				t.Fatalf("service id = %q, want %q", body.Services[0].ID, serviceID.String())
			}
			if body.Services[0].Name != tt.wantName {
				t.Fatalf("service name = %q, want %q", body.Services[0].Name, tt.wantName)
			}
		})
	}
}
