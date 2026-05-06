package auth

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	authmw "github.com/mams/backend/internal/middleware/auth"
	"github.com/mams/backend/internal/models"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
	"github.com/mams/backend/internal/handlers/auth/mocks"
	"github.com/mams/backend/internal/logx"
	"go.uber.org/mock/gomock"
)

func TestLoginHandler_Me(t *testing.T) {
	t.Parallel()

	type want struct {
		code int
		body map[string]any
	}

	cases := []struct {
		name  string
		setup func(m *mocks.MockUserReader) *http.Request
		want  want
	}{
		{
			name: "ok",
			setup: func(m *mocks.MockUserReader) *http.Request {
				userID := uuid.MustParse("a08f1e57-df6a-4f31-bfd1-73dc497d1820")
				orgID := uuid.MustParse("64caed96-34db-4822-8de0-d77d4bb6be43")
				m.EXPECT().GetByID(gomock.Any(), userID).Return(models.User{
					ID:             userID,
					Login:          "admin",
					OrganizationID: orgID,
				}, nil)
				m.EXPECT().ListUserNonObserverRoles(gomock.Any(), userID, orgID).Return([]models.ProfileServiceRole{
					{
						ServiceID:   uuid.MustParse("f4b21e0e-31bd-4d3f-a607-d7af0f0f8f8e"),
						ServiceName: "user-service",
						Role:        "developer",
					},
				}, nil)
				req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
				req = req.WithContext(authmw.WithClaims(req.Context(), authmw.Claims{
					UserID:         userID,
					OrganizationID: orgID,
				}))
				return req
			},
			want: want{
				code: http.StatusOK,
				body: map[string]any{
					"id":              "a08f1e57-df6a-4f31-bfd1-73dc497d1820",
					"login":           "admin",
					"organization_id": "64caed96-34db-4822-8de0-d77d4bb6be43",
					"services": []any{
						map[string]any{
							"service_id":   "f4b21e0e-31bd-4d3f-a607-d7af0f0f8f8e",
							"service_name": "user-service",
							"role":         "developer",
						},
					},
				},
			},
		},
		{
			name: "no claims",
			setup: func(m *mocks.MockUserReader) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
			},
			want: want{
				code: http.StatusUnauthorized,
				body: map[string]any{"error": "unauthorized"},
			},
		},
		{
			name: "not found",
			setup: func(m *mocks.MockUserReader) *http.Request {
				userID := uuid.MustParse("0c487ca6-9d2f-4375-9f5a-4cc4d9dc85ec")
				m.EXPECT().GetByID(gomock.Any(), userID).Return(models.User{}, postgresrepo.ErrUserNotFound)
				req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
				req = req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{UserID: userID}))
				return req
			},
			want: want{
				code: http.StatusUnauthorized,
				body: map[string]any{"error": "unauthorized"},
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			users := mocks.NewMockUserReader(ctrl)
			h := NewLoginHandler(users, nil, logx.New(slog.New(slog.NewTextHandler(io.Discard, nil))))
			req := tc.setup(users)
			rec := httptest.NewRecorder()

			h.Me(rec, req)

			if rec.Code != tc.want.code {
				t.Fatalf("status code = %d, want %d", rec.Code, tc.want.code)
			}

			var got map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal body: %v", err)
			}
			if len(got) != len(tc.want.body) {
				t.Fatalf("body len = %d, want %d", len(got), len(tc.want.body))
			}
			for k, v := range tc.want.body {
				if k == "services" {
					gotServices, ok := got["services"].([]any)
					if !ok || len(gotServices) != 1 {
						t.Fatalf("services invalid: %v", got["services"])
					}
					item, ok := gotServices[0].(map[string]any)
					if !ok {
						t.Fatalf("service item invalid: %T", gotServices[0])
					}
					wantItem := v.([]any)[0].(map[string]any)
					for wk, wv := range wantItem {
						if item[wk] != wv {
							t.Fatalf("services[0][%q] = %v, want %v", wk, item[wk], wv)
						}
					}
					continue
				}
				if got[k] != v {
					t.Fatalf("body[%q] = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}
