package releases

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/mams/backend/internal/handlers/releases/mocks"
	authmw "github.com/mams/backend/internal/middleware/auth"
	"github.com/mams/backend/internal/models"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
	"go.uber.org/mock/gomock"
)

func TestGet_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	rr := mocks.NewMockReleaseReader(ctrl)
	org := uuid.New()
	sid := uuid.New()
	sr.EXPECT().GetByID(gomock.Any(), sid).Return(models.Service{ID: sid, OrganizationID: org}, nil)
	rr.EXPECT().ListByService(gomock.Any(), sid).Return([]models.Release{{ID: uuid.New(), ServiceID: sid}}, nil)
	h := NewHandler(sr, rr)
	req := httptest.NewRequest(http.MethodGet, "/api/services/"+sid.String()+"/releases", nil)
	req.SetPathValue("id", sid.String())
	req = req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: org}))
	rec := httptest.NewRecorder()
	h.Get(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestCreatePending_Status(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	rr := mocks.NewMockReleaseReader(ctrl)
	sid := uuid.New()
	uid := uuid.New()
	rr.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, rel models.Release) (models.Release, error) {
		if rel.Status != "pending" {
			t.Fatalf("status=%s", rel.Status)
		}
		return rel, nil
	})
	h := NewHandler(sr, rr)
	if _, err := h.CreatePending(context.Background(), sid, uid, "v1", "main", "prod", "rolling", "desc"); err != nil {
		t.Fatalf("err=%v", err)
	}
}

func TestGet_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	rr := mocks.NewMockReleaseReader(ctrl)
	sid := uuid.New()
	sr.EXPECT().GetByID(gomock.Any(), sid).Return(models.Service{}, postgresrepo.ErrServiceNotFound)
	h := NewHandler(sr, rr)
	req := httptest.NewRequest(http.MethodGet, "/api/services/"+sid.String()+"/releases", nil)
	req.SetPathValue("id", sid.String())
	req = req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: uuid.New()}))
	rec := httptest.NewRecorder()
	h.Get(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

