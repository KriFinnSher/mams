package releases

import (
	"context"
	"bytes"
	"encoding/json"
	"errors"
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
	h := NewHandler(sr, rr, &testWorkflowDispatcher{}, nil)
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
	h := NewHandler(sr, rr, &testWorkflowDispatcher{}, nil)
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
	h := NewHandler(sr, rr, &testWorkflowDispatcher{}, nil)
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
	h := NewHandler(sr, rr, &testWorkflowDispatcher{}, nil)
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
	h := NewHandler(sr, rr, &testWorkflowDispatcher{}, nil)
	if _, err := h.MarkResult(context.Background(), rid, "success"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMarkResult_InvalidTargetStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	rr := mocks.NewMockReleaseReader(ctrl)
	h := NewHandler(sr, rr, &testWorkflowDispatcher{}, nil)
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
	h := NewHandler(sr, rr, &testWorkflowDispatcher{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/services/"+sid.String()+"/releases", nil)
	req.SetPathValue("id", sid.String())
	req = req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: uuid.New()}))
	rec := httptest.NewRecorder()
	h.Get(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestRollbackCandidates_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	rr := mocks.NewMockReleaseReader(ctrl)
	org := uuid.New()
	sid := uuid.New()
	sr.EXPECT().GetByID(gomock.Any(), sid).Return(models.Service{ID: sid, OrganizationID: org}, nil)
	rr.EXPECT().ListByService(gomock.Any(), sid).Return([]models.Release{
		{ServiceID: sid, Status: "success", GitTag: "v1.0.3"},
		{ServiceID: sid, Status: "failed", GitTag: "v1.0.2"},
		{ServiceID: sid, Status: "success", GitTag: "v1.0.1"},
		{ServiceID: sid, Status: "success", GitTag: "v1.0.3"},
	}, nil)
	h := NewHandler(sr, rr, &testWorkflowDispatcher{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/services/"+sid.String()+"/rollback/candidates", nil)
	req.SetPathValue("id", sid.String())
	req = req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: org}))
	rec := httptest.NewRecorder()
	h.RollbackCandidates(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	var body map[string][]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(body["git_tags"]) != 2 {
		t.Fatalf("git_tags len=%d", len(body["git_tags"]))
	}
}

func TestRollbackCandidates_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	rr := mocks.NewMockReleaseReader(ctrl)
	sid := uuid.New()
	sr.EXPECT().GetByID(gomock.Any(), sid).Return(models.Service{}, postgresrepo.ErrServiceNotFound)
	h := NewHandler(sr, rr, &testWorkflowDispatcher{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/services/"+sid.String()+"/rollback/candidates", nil)
	req.SetPathValue("id", sid.String())
	req = req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: uuid.New()}))
	rec := httptest.NewRecorder()
	h.RollbackCandidates(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestRollbackCandidates_ListError(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	rr := mocks.NewMockReleaseReader(ctrl)
	org := uuid.New()
	sid := uuid.New()
	sr.EXPECT().GetByID(gomock.Any(), sid).Return(models.Service{ID: sid, OrganizationID: org}, nil)
	rr.EXPECT().ListByService(gomock.Any(), sid).Return(nil, errors.New("db error"))
	h := NewHandler(sr, rr, &testWorkflowDispatcher{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/services/"+sid.String()+"/rollback/candidates", nil)
	req.SetPathValue("id", sid.String())
	req = req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: org}))
	rec := httptest.NewRecorder()
	h.RollbackCandidates(rec, req)
	if rec.Code != http.StatusInternalServerError {
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
	h := NewHandler(sr, rr, wd, nil)
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
	h := NewHandler(sr, rr, wd, nil)
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
	h := NewHandler(sr, rr, wd, nil)

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
	h := NewHandler(sr, rr, wd, nil)
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
	h := NewHandler(sr, rr, wd, nil)
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
	h := NewHandler(sr, rr, wd, nil)
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

func TestRollback_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	rr := mocks.NewMockReleaseReader(ctrl)
	wd := &testWorkflowDispatcher{}
	org := uuid.New()
	sid := uuid.New()
	uid := uuid.New()
	sr.EXPECT().GetByID(gomock.Any(), sid).Return(models.Service{
		ID: sid, OrganizationID: org, RepositoryURL: "https://github.com/acme/repo",
	}, nil)
	rr.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, rel models.Release) (models.Release, error) {
		if rel.Strategy != "rollback" {
			t.Fatalf("strategy=%s", rel.Strategy)
		}
		rel.ID = uuid.New()
		rel.Status = "pending"
		return rel, nil
	})
	rr.EXPECT().UpdateStatus(gomock.Any(), gomock.Any(), "in_progress").DoAndReturn(func(_ context.Context, id uuid.UUID, status string) (models.Release, error) {
		return models.Release{ID: id, ServiceID: sid, Status: status}, nil
	})
	h := NewHandler(sr, rr, wd, nil)
	body, _ := json.Marshal(map[string]any{"git_tag": "v1.2.1", "environment": "prod", "description": "rollback"})
	req := httptest.NewRequest(http.MethodPost, "/api/services/"+sid.String()+"/rollback", bytes.NewReader(body))
	req.SetPathValue("id", sid.String())
	req = req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: org, UserID: uid}))
	rec := httptest.NewRecorder()
	h.Rollback(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d", rec.Code)
	}
	if !wd.called {
		t.Fatalf("workflow not called")
	}
}

func TestRollback_RequiresGitTag(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	rr := mocks.NewMockReleaseReader(ctrl)
	wd := &testWorkflowDispatcher{}
	org := uuid.New()
	sid := uuid.New()
	sr.EXPECT().GetByID(gomock.Any(), sid).Return(models.Service{
		ID: sid, OrganizationID: org, RepositoryURL: "https://github.com/acme/repo",
	}, nil)
	h := NewHandler(sr, rr, wd, nil)
	body, _ := json.Marshal(map[string]any{"git_tag": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/services/"+sid.String()+"/rollback", bytes.NewReader(body))
	req.SetPathValue("id", sid.String())
	req = req.WithContext(authmw.WithClaims(context.Background(), authmw.Claims{OrganizationID: org, UserID: uuid.New()}))
	rec := httptest.NewRecorder()
	h.Rollback(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestUpdateFromCI_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	rr := mocks.NewMockReleaseReader(ctrl)
	rid := uuid.New()
	sid := uuid.New()
	rr.EXPECT().GetByID(gomock.Any(), rid).Return(models.Release{ID: rid, Status: "in_progress"}, nil)
	rr.EXPECT().UpdateStatus(gomock.Any(), rid, "success").Return(models.Release{
		ID: rid, ServiceID: sid, GitTag: "v1.0.0", Status: "success",
	}, nil)
	rr.EXPECT().UpdateServiceVersion(gomock.Any(), sid, "v1.0.0").Return(nil)
	h := NewHandler(sr, rr, &testWorkflowDispatcher{}, nil)
	body, _ := json.Marshal(map[string]any{"release_id": rid.String(), "status": "success"})
	req := httptest.NewRequest(http.MethodPost, "/api/internal/releases/status", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.UpdateFromCI(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestUpdateFromCI_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	rr := mocks.NewMockReleaseReader(ctrl)
	rid := uuid.New()
	rr.EXPECT().GetByID(gomock.Any(), rid).Return(models.Release{}, postgresrepo.ErrReleaseNotFound)
	h := NewHandler(sr, rr, &testWorkflowDispatcher{}, nil)
	body, _ := json.Marshal(map[string]any{"release_id": rid.String(), "status": "failed"})
	req := httptest.NewRequest(http.MethodPost, "/api/internal/releases/status", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.UpdateFromCI(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}
