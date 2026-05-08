package releases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	authmw "github.com/mams/backend/internal/middleware/auth"
	"github.com/mams/backend/internal/models"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
	"github.com/mams/backend/internal/utils"
)

type Handler struct {
	services ServiceReader
	releases ReleaseReader
	workflows WorkflowDispatcher
}

func NewHandler(services ServiceReader, releases ReleaseReader, workflows WorkflowDispatcher) *Handler {
	return &Handler{services: services, releases: releases, workflows: workflows}
}

func (h *Handler) CreatePending(
	ctx context.Context,
	serviceID, authorUserID uuid.UUID,
	gitTag, branch, environment, strategy, description string,
) (models.Release, error) {
	return h.releases.Create(ctx, models.Release{
		ServiceID:    serviceID,
		GitTag:       gitTag,
		Branch:       branch,
		Environment:  environment,
		Strategy:     strategy,
		Status:       "pending",
		Description:  description,
		AuthorUserID: authorUserID,
	})
}

func (h *Handler) MarkResult(ctx context.Context, releaseID uuid.UUID, status string) (models.Release, error) {
	if status != "success" && status != "failed" {
		return models.Release{}, fmt.Errorf("invalid target status: %s", status)
	}
	current, err := h.releases.GetByID(ctx, releaseID)
	if err != nil {
		return models.Release{}, err
	}
	if current.Status != "in_progress" {
		return models.Release{}, fmt.Errorf("invalid current status: %s", current.Status)
	}

	updated, err := h.releases.UpdateStatus(ctx, releaseID, status)
	if err != nil {
		return models.Release{}, err
	}
	if status == "success" && updated.GitTag != "" {
		if err := h.releases.UpdateServiceVersion(ctx, updated.ServiceID, updated.GitTag); err != nil {
			return models.Release{}, err
		}
	}

	return updated, nil
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	claims, ok := authmw.ClaimsFromContext(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}
	svc, err := h.services.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, postgresrepo.ErrServiceNotFound) {
			utils.WriteError(w, http.StatusNotFound, "service not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if svc.OrganizationID != claims.OrganizationID {
		utils.WriteError(w, http.StatusNotFound, "service not found")
		return
	}
	list, err := h.releases.ListByService(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	resp := make([]map[string]any, 0, len(list))
	for _, rel := range list {
		resp = append(resp, map[string]any{
			"id":             rel.ID.String(),
			"service_id":     rel.ServiceID.String(),
			"git_tag":        rel.GitTag,
			"branch":         rel.Branch,
			"environment":    rel.Environment,
			"strategy":       rel.Strategy,
			"status":         rel.Status,
			"description":    rel.Description,
			"author_user_id": rel.AuthorUserID.String(),
			"deployed_at":    rel.DeployedAt,
		})
	}
	utils.WriteJSON(w, http.StatusOK, map[string]any{"releases": resp})
}

func (h *Handler) RollbackCandidates(w http.ResponseWriter, r *http.Request) {
	claims, ok := authmw.ClaimsFromContext(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}
	svc, err := h.services.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, postgresrepo.ErrServiceNotFound) {
			utils.WriteError(w, http.StatusNotFound, "service not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if svc.OrganizationID != claims.OrganizationID {
		utils.WriteError(w, http.StatusNotFound, "service not found")
		return
	}
	list, err := h.releases.ListByService(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	seen := make(map[string]struct{}, len(list))
	tags := make([]string, 0, len(list))
	for _, rel := range list {
		if rel.Status != "success" || rel.GitTag == "" {
			continue
		}
		if _, ok := seen[rel.GitTag]; ok {
			continue
		}
		seen[rel.GitTag] = struct{}{}
		tags = append(tags, rel.GitTag)
	}
	utils.WriteJSON(w, http.StatusOK, map[string]any{"git_tags": tags})
}

type deployRequest struct {
	GitTag      string `json:"git_tag"`
	Branch      string `json:"branch"`
	Environment string `json:"environment"`
	Strategy    string `json:"strategy"`
	Description string `json:"description"`
}

type rollbackRequest struct {
	GitTag      string `json:"git_tag"`
	Environment string `json:"environment"`
	Description string `json:"description"`
}

type ciStatusRequest struct {
	ReleaseID string `json:"release_id"`
	Status    string `json:"status"`
}

func (h *Handler) Deploy(w http.ResponseWriter, r *http.Request) {
	claims, ok := authmw.ClaimsFromContext(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}
	svc, err := h.services.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, postgresrepo.ErrServiceNotFound) {
			utils.WriteError(w, http.StatusNotFound, "service not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if svc.OrganizationID != claims.OrganizationID {
		utils.WriteError(w, http.StatusNotFound, "service not found")
		return
	}
	if h.workflows == nil {
		utils.WriteError(w, http.StatusInternalServerError, "workflow dispatcher is not configured")
		return
	}

	var req deployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	originalBranch := req.Branch
	if req.Branch == "" && req.GitTag == "" {
		req.Branch = svc.DefaultBranch
	}
	if req.Environment == "" {
		req.Environment = "dev"
	}
	if req.Strategy == "" {
		req.Strategy = "rolling"
	}
	if req.Environment == "prod" && req.GitTag == "" {
		utils.WriteError(w, http.StatusBadRequest, "git_tag is required for prod deploy")
		return
	}
	if (req.Environment == "dev" || req.Environment == "staging") && originalBranch == "" {
		utils.WriteError(w, http.StatusBadRequest, "branch is required for dev/staging deploy")
		return
	}
	if svc.MinimumTestCoverageEnabled && svc.TestCoverage < svc.MinimumTestCoverage {
		utils.WriteError(w, http.StatusBadRequest, "minimum test coverage requirement is not met")
		return
	}

	created, err := h.CreatePending(r.Context(), svc.ID, claims.UserID, req.GitTag, req.Branch, req.Environment, req.Strategy, req.Description)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	ref := req.Branch
	if ref == "" {
		ref = req.GitTag
	}
	inputs := map[string]string{
		"service_id":   svc.ID.String(),
		"environment":  req.Environment,
		"strategy":     req.Strategy,
		"description":  req.Description,
		"release_id":   created.ID.String(),
	}
	if req.GitTag != "" {
		inputs["git_tag"] = req.GitTag
	}
	if req.Branch != "" {
		inputs["branch"] = req.Branch
	}

	if err := h.workflows.DispatchWorkflow(r.Context(), svc.RepositoryURL, "deploy.yml", ref, inputs); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "failed to dispatch workflow")
		return
	}
	inProgress, err := h.releases.UpdateStatus(r.Context(), created.ID, "in_progress")
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	utils.WriteJSON(w, http.StatusAccepted, map[string]any{
		"release_id": inProgress.ID.String(),
		"status":     inProgress.Status,
	})
}

func (h *Handler) Rollback(w http.ResponseWriter, r *http.Request) {
	claims, ok := authmw.ClaimsFromContext(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}
	svc, err := h.services.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, postgresrepo.ErrServiceNotFound) {
			utils.WriteError(w, http.StatusNotFound, "service not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if svc.OrganizationID != claims.OrganizationID {
		utils.WriteError(w, http.StatusNotFound, "service not found")
		return
	}
	if h.workflows == nil {
		utils.WriteError(w, http.StatusInternalServerError, "workflow dispatcher is not configured")
		return
	}

	var req rollbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.GitTag == "" {
		utils.WriteError(w, http.StatusBadRequest, "git_tag is required for rollback")
		return
	}
	if req.Environment == "" {
		req.Environment = "prod"
	}

	created, err := h.CreatePending(r.Context(), svc.ID, claims.UserID, req.GitTag, "", req.Environment, "rollback", req.Description)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	inputs := map[string]string{
		"service_id":  svc.ID.String(),
		"environment": req.Environment,
		"strategy":    "rollback",
		"description": req.Description,
		"release_id":  created.ID.String(),
		"git_tag":     req.GitTag,
		"rollback":    "true",
	}
	if err := h.workflows.DispatchWorkflow(r.Context(), svc.RepositoryURL, "deploy.yml", req.GitTag, inputs); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "failed to dispatch workflow")
		return
	}
	inProgress, err := h.releases.UpdateStatus(r.Context(), created.ID, "in_progress")
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	utils.WriteJSON(w, http.StatusAccepted, map[string]any{
		"release_id": inProgress.ID.String(),
		"status":     inProgress.Status,
	})
}

func (h *Handler) UpdateFromCI(w http.ResponseWriter, r *http.Request) {
	var req ciStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	releaseID, err := uuid.Parse(req.ReleaseID)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid release id")
		return
	}
	updated, err := h.MarkResult(r.Context(), releaseID, req.Status)
	if err != nil {
		if errors.Is(err, postgresrepo.ErrReleaseNotFound) {
			utils.WriteError(w, http.StatusNotFound, "release not found")
			return
		}
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{
		"release_id": updated.ID.String(),
		"status":     updated.Status,
	})
}
