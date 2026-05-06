package rbac

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	authmw "github.com/mams/backend/internal/middleware/auth"
)

func TestRequireSettingsAccess(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	serviceID := uuid.New()

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	h := RequireSettingsAccess(
		testServiceReader{svc: serviceView{OrganizationID: orgID, OwnerUserID: userID}},
		testAccessReader{err: ErrAccessNotFound},
		next,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/services/"+serviceID.String()+"/settings", nil)
	req.SetPathValue("id", serviceID.String())
	req = req.WithContext(authmw.WithClaims(req.Context(), authmw.Claims{
		UserID:         userID,
		OrganizationID: orgID,
	}))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

