package metrics

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	authmw "github.com/mams/backend/internal/middleware/auth"
	"github.com/mams/backend/internal/models"
	"github.com/mams/backend/internal/repository/postgres"
	"github.com/mams/backend/internal/handlers/metrics/mocks"
	"go.uber.org/mock/gomock"
)

func TestGet(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	org := uuid.New()
	sid := uuid.New()
	sr.EXPECT().GetByID(gomock.Any(), sid).Return(models.Service{ID: sid, OrganizationID: org, GrafanaDashboardUID: "uid1"}, nil)
	h := NewHandler(sr, "https://grafana.local")
	req := httptest.NewRequest(http.MethodGet, "/api/services/"+sid.String()+"/metrics", nil)
	req.SetPathValue("id", sid.String())
	req = req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: org}))
	rec := httptest.NewRecorder()
	h.Get(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["embed_url"] != "https://grafana.local/d/uid1?kiosk" {
		t.Fatalf("embed_url=%v", body["embed_url"])
	}
}

func TestGet_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	sid := uuid.New()
	sr.EXPECT().GetByID(gomock.Any(), sid).Return(models.Service{}, postgres.ErrServiceNotFound)
	h := NewHandler(sr, "https://grafana.local")
	req := httptest.NewRequest(http.MethodGet, "/api/services/"+sid.String()+"/metrics", nil)
	req.SetPathValue("id", sid.String())
	req = req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: uuid.New()}))
	rec := httptest.NewRecorder()
	h.Get(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

