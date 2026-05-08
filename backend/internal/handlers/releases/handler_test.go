package releases

import (
	"context"
	"bytes"
	"encoding/json"
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

type testWorkflowDispatcher struct {
	called bool
	err    error
}

func (d *testWorkflowDispatcher) DispatchWorkflow(_ context.Context, _, _, _ string, _ map[string]string) error {
	d.called = true
	return d.err
}

func TestGet_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	rr := mocks.NewMockReleaseReader(ctrl)
	org := uuid.New()
	sid := uuid.New()
	sr.EXPECT().GetByID(gomock.Any(), sid).Return(models.Service{ID: sid, OrganizationID: org}, nil)
	rr.EXPECT().ListByService(gomock.Any(), sid).Return([]models.Release{{ID: uuid.New(), ServiceID: sid}}, nil)
	h := NewHandler(sr, rr, &testWorkflowDispatcher{})
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
	h := NewHandler(sr, rr, &testWorkflowDispatcher{})
	if _, err := h.CreatePending(context.Background(), sid, uid, "v1", "main", "prod", "rolling", "desc"); err != nil {
		t.Fatalf("err=%v", err)
	}
}

func TestMarkResult_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	rr := mocks.NewMockReleaseReader(ctrl)
	rid := uuid.New()
	rr.EXPECT().GetByID(gomock.Any(), rid).Return(models.Release{ID: rid, Status: "in_progress"}, nil)
	sid := uuid.New()
	rr.EXPECT().UpdateStatus(gomock.Any(), rid, "success").Return(models.Release{ID: rid, ServiceID: sid, GitTag: "v1.2.3", Status: "success"}, nil)
	rr.EXPECT().UpdateServiceVersion(gomock.Any(), sid, "v1.2.3").Return(nil)
	h := NewHandler(sr, rr, &testWorkflowDispatcher{})
	got, err := h.MarkResult(context.Background(), rid, "success")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got.Status != "success" {
		t.Fatalf("status=%s", got.Status)
	}
}

func TestMarkResult_FailedDoesNotUpdateServiceVersion(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	rr := mocks.NewMockReleaseReader(ctrl)
	rid := uuid.New()
	rr.EXPECT().GetByID(gomock.Any(), rid).Return(models.Release{ID: rid, Status: "in_progress"}, nil)
	rr.EXPECT().UpdateStatus(gomock.Any(), rid, "failed").Return(models.Release{ID: rid, Status: "failed"}, nil)
	h := NewHandler(sr, rr, &testWorkflowDispatcher{})
	got, err := h.MarkResult(context.Background(), rid, "failed")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got.Status != "failed" {
		t.Fatalf("status=%s", got.Status)
	}
}

func TestMarkResult_InvalidTransition(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	rr := mocks.NewMockReleaseReader(ctrl)
	rid := uuid.New()
	rr.EXPECT().GetByID(gomock.Any(), rid).Return(models.Release{ID: rid, Status: "pending"}, nil)
	h := NewHandler(sr, rr, &testWorkflowDispatcher{})
	if _, err := h.MarkResult(context.Background(), rid, "success"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMarkResult_InvalidTargetStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	rr := mocks.NewMockReleaseReader(ctrl)
	h := NewHandler(sr, rr, &testWorkflowDispatcher{})
	if _, err := h.MarkResult(context.Background(), uuid.New(), "pending"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestGet_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	rr := mocks.NewMockReleaseReader(ctrl)
	sid := uuid.New()
	sr.EXPECT().GetByID(gomock.Any(), sid).Return(models.Service{}, postgresrepo.ErrServiceNotFound)
	h := NewHandler(sr, rr, &testWorkflowDispatcher{})
	req := httptest.NewRequest(http.MethodGet, "/api/services/"+sid.String()+"/releases", nil)
	req.SetPathValue("id", sid.String())
	req = req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: uuid.New()}))
	rec := httptest.NewRecorder()
	h.Get(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestDeploy_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	rr := mocks.NewMockReleaseReader(ctrl)
	wd := &testWorkflowDispatcher{}
	org := uuid.New()
	sid := uuid.New()
	uid := uuid.New()
	sr.EXPECT().GetByID(gomock.Any(), sid).Return(models.Service{
		ID: sid, OrganizationID: org, RepositoryURL: "https://github.com/acme/repo", DefaultBranch: "main",
	}, nil)
	rr.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, rel models.Release) (models.Release, error) {
		rel.ID = uuid.New()
		rel.Status = "pending"
		if rel.AuthorUserID != uid {
			t.Fatalf("author mismatch")
		}
		return rel, nil
	})
	rr.EXPECT().UpdateStatus(gomock.Any(), gomock.Any(), "in_progress").DoAndReturn(func(_ context.Context, id uuid.UUID, status string) (models.Release, error) {
		return models.Release{ID: id, ServiceID: sid, Status: status}, nil
	})
	h := NewHandler(sr, rr, wd)
	body, _ := json.Marshal(map[string]any{"environment": "dev", "strategy": "rolling", "branch": "main"})
	req := httptest.NewRequest(http.MethodPost, "/api/services/"+sid.String()+"/deploy", bytes.NewReader(body))
	req.SetPathValue("id", sid.String())
	req = req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: org, UserID: uid}))
	rec := httptest.NewRecorder()
	h.Deploy(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d", rec.Code)
	}
	if !wd.called {
		t.Fatalf("workflow not called")
	}
}

func TestDeploy_InvalidBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	rr := mocks.NewMockReleaseReader(ctrl)
	wd := &testWorkflowDispatcher{}
	org := uuid.New()
	sid := uuid.New()
	sr.EXPECT().GetByID(gomock.Any(), sid).Return(models.Service{
		ID: sid, OrganizationID: org, RepositoryURL: "https://github.com/acme/repo", DefaultBranch: "main",
	}, nil)
	h := NewHandler(sr, rr, wd)
	req := httptest.NewRequest(http.MethodPost, "/api/services/"+sid.String()+"/deploy", bytes.NewReader([]byte("{")))
	req.SetPathValue("id", sid.String())
	req = req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: org, UserID: uuid.New()}))
	rec := httptest.NewRecorder()
	h.Deploy(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestDeploy_BranchRequiredForDevStaging(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	rr := mocks.NewMockReleaseReader(ctrl)
	wd := &testWorkflowDispatcher{}
	org := uuid.New()
	sid := uuid.New()
	sr.EXPECT().GetByID(gomock.Any(), sid).Return(models.Service{
		ID: sid, OrganizationID: org, RepositoryURL: "https://github.com/acme/repo", DefaultBranch: "main",
	}, nil).Times(2)
	h := NewHandler(sr, rr, wd)

	cases := []string{"dev", "staging"}
	for _, env := range cases {
		body, _ := json.Marshal(map[string]any{"environment": env, "strategy": "rolling", "branch": ""})
		req := httptest.NewRequest(http.MethodPost, "/api/services/"+sid.String()+"/deploy", bytes.NewReader(body))
		req.SetPathValue("id", sid.String())
		req = req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: org, UserID: uuid.New()}))
		rec := httptest.NewRecorder()
		h.Deploy(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("env=%s status=%d", env, rec.Code)
		}
	}
}

func TestDeploy_UsesDefaultBranchFallbackForNonDevStaging(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	rr := mocks.NewMockReleaseReader(ctrl)
	wd := &testWorkflowDispatcher{}
	org := uuid.New()
	sid := uuid.New()
	uid := uuid.New()
	sr.EXPECT().GetByID(gomock.Any(), sid).Return(models.Service{
		ID: sid, OrganizationID: org, RepositoryURL: "https://github.com/acme/repo", DefaultBranch: "main",
	}, nil)
	rr.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, rel models.Release) (models.Release, error) {
		if rel.Branch != "main" {
			t.Fatalf("expected default branch fallback, got %q", rel.Branch)
		}
		return models.Release{ID: uuid.New(), ServiceID: sid, Status: "pending"}, nil
	})
	rr.EXPECT().UpdateStatus(gomock.Any(), gomock.Any(), "in_progress").DoAndReturn(func(_ context.Context, id uuid.UUID, status string) (models.Release, error) {
		return models.Release{ID: id, ServiceID: sid, Status: status}, nil
	})
	h := NewHandler(sr, rr, wd)
	body, _ := json.Marshal(map[string]any{"environment": "qa", "strategy": "rolling", "branch": "", "git_tag": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/services/"+sid.String()+"/deploy", bytes.NewReader(body))
	req.SetPathValue("id", sid.String())
	req = req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: org, UserID: uid}))
	rec := httptest.NewRecorder()
	h.Deploy(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d", rec.Code)
	}
	if !wd.called {
		t.Fatalf("workflow not called")
	}
}

func TestDeploy_GitTagRequiredForProd(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	rr := mocks.NewMockReleaseReader(ctrl)
	wd := &testWorkflowDispatcher{}
	org := uuid.New()
	sid := uuid.New()
	sr.EXPECT().GetByID(gomock.Any(), sid).Return(models.Service{
		ID: sid, OrganizationID: org, RepositoryURL: "https://github.com/acme/repo", DefaultBranch: "main",
	}, nil)
	h := NewHandler(sr, rr, wd)
	body, _ := json.Marshal(map[string]any{"environment": "prod", "strategy": "rolling", "branch": "main", "git_tag": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/services/"+sid.String()+"/deploy", bytes.NewReader(body))
	req.SetPathValue("id", sid.String())
	req = req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: org, UserID: uuid.New()}))
	rec := httptest.NewRecorder()
	h.Deploy(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestDeploy_BlocksWhenCoverageRequirementIsNotMet(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	rr := mocks.NewMockReleaseReader(ctrl)
	wd := &testWorkflowDispatcher{}
	org := uuid.New()
	sid := uuid.New()
	sr.EXPECT().GetByID(gomock.Any(), sid).Return(models.Service{
		ID: sid, OrganizationID: org, RepositoryURL: "https://github.com/acme/repo", DefaultBranch: "main",
		TestCoverage: 65, MinimumTestCoverageEnabled: true, MinimumTestCoverage: 70,
	}, nil)
	h := NewHandler(sr, rr, wd)
	body, _ := json.Marshal(map[string]any{"environment": "dev", "strategy": "rolling", "branch": "main"})
	req := httptest.NewRequest(http.MethodPost, "/api/services/"+sid.String()+"/deploy", bytes.NewReader(body))
	req.SetPathValue("id", sid.String())
	req = req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: org, UserID: uuid.New()}))
	rec := httptest.NewRecorder()
	h.Deploy(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
	if wd.called {
		t.Fatalf("workflow should not be called")
	}
}
