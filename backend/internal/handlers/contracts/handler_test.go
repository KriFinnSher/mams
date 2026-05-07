package contracts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/mams/backend/internal/handlers/contracts/mocks"
	authmw "github.com/mams/backend/internal/middleware/auth"
	"github.com/mams/backend/internal/models"
	"github.com/mams/backend/internal/repository/postgres"
	"go.uber.org/mock/gomock"
)

func TestGet_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	pr := mocks.NewMockProtoReader(ctrl)
	org := uuid.New()
	sid := uuid.New()
	sr.EXPECT().GetByID(gomock.Any(), sid).Return(models.Service{ID: sid, OrganizationID: org, RepositoryURL: "https://github.com/acme/repo", DefaultBranch: "main"}, nil)
	pr.EXPECT().ReadProjectProto(gomock.Any(), "https://github.com/acme/repo", "main").Return([]byte(`service S { rpc Get(GetReq) returns (GetRes); } message GetReq { string id = 1; }`), nil)
	h := NewHandler(sr, pr)
	req := httptest.NewRequest(http.MethodGet, "/api/services/"+sid.String()+"/contracts", nil)
	req.SetPathValue("id", sid.String())
	req = req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: org}))
	rec := httptest.NewRecorder()
	h.Get(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestGet_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	pr := mocks.NewMockProtoReader(ctrl)
	sid := uuid.New()
	sr.EXPECT().GetByID(gomock.Any(), sid).Return(models.Service{}, postgres.ErrServiceNotFound)
	h := NewHandler(sr, pr)
	req := httptest.NewRequest(http.MethodGet, "/api/services/"+sid.String()+"/contracts", nil)
	req.SetPathValue("id", sid.String())
	req = req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: uuid.New()}))
	rec := httptest.NewRecorder()
	h.Get(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

