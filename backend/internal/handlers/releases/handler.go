package releases

import (
	"context"
	"encoding/json"
	"errors"
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

type deployRequest struct {
	GitTag      string `json:"git_tag"`
	Branch      string `json:"branch"`
	Environment string `json:"environment"`
	Strategy    string `json:"strategy"`
	Description string `json:"description"`
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
	if req.Branch == "" && req.GitTag == "" {
		req.Branch = svc.DefaultBranch
	}
	if req.Environment == "" {
		req.Environment = "dev"
	}
	if req.Strategy == "" {
		req.Strategy = "rolling"
	}
	if (req.Environment == "dev" || req.Environment == "staging") && req.Branch == "" {
		utils.WriteError(w, http.StatusBadRequest, "branch is required for dev/staging deploy")
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

	utils.WriteJSON(w, http.StatusAccepted, map[string]any{
		"release_id": created.ID.String(),
		"status":     created.Status,
	})
}
