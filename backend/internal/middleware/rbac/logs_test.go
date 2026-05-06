package rbac

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	authmw "github.com/mams/backend/internal/middleware/auth"
)

type testServiceReader struct {
	svc serviceView
	err error
}

func (t testServiceReader) GetByID(context.Context, uuid.UUID) (serviceView, error) {
	return t.svc, t.err
}

type testAccessReader struct {
	acc accessView
	err error
}

func (t testAccessReader) GetByServiceAndUser(context.Context, uuid.UUID, uuid.UUID) (accessView, error) {
	return t.acc, t.err
}

func TestRequireLogsAccess(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	serviceID := uuid.New()

	tests := []struct {
		name       string
		claims     *authmw.Claims
		services   serviceReader
		access     accessReader
		wantStatus int
	}{
		{
			name:       "no claims",
			services:   testServiceReader{},
			access:     testAccessReader{},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "owner allowed",
			claims: &authmw.Claims{
				UserID:         userID,
				OrganizationID: orgID,
			},
			services: testServiceReader{svc: serviceView{OrganizationID: orgID, OwnerUserID: userID}},
			access:   testAccessReader{err: ErrAccessNotFound},
			wantStatus: http.StatusNoContent,
		},
		{
			name: "developer allowed",
			claims: &authmw.Claims{
				UserID:         userID,
				OrganizationID: orgID,
			},
			services: testServiceReader{svc: serviceView{OrganizationID: orgID, OwnerUserID: uuid.New()}},
			access:   testAccessReader{acc: accessView{Role: "developer"}},
			wantStatus: http.StatusNoContent,
		},
		{
			name: "observer denied",
			claims: &authmw.Claims{
				UserID:         userID,
				OrganizationID: orgID,
			},
			services: testServiceReader{svc: serviceView{OrganizationID: orgID, OwnerUserID: uuid.New()}},
			access:   testAccessReader{err: ErrAccessNotFound},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "access backend error",
			claims: &authmw.Claims{
				UserID:         userID,
				OrganizationID: orgID,
			},
			services: testServiceReader{svc: serviceView{OrganizationID: orgID, OwnerUserID: uuid.New()}},
			access:   testAccessReader{err: errors.New("db")},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})
			h := RequireLogsAccess(tt.services, tt.access, next)
			req := httptest.NewRequest(http.MethodGet, "/api/services/"+serviceID.String()+"/logs", nil)
			req.SetPathValue("id", serviceID.String())
			if tt.claims != nil {
				req = req.WithContext(authmw.WithClaims(req.Context(), *tt.claims))
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
