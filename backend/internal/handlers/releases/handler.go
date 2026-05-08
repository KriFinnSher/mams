package releases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	authmw "github.com/mams/backend/internal/middleware/auth"
	"github.com/mams/backend/internal/models"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
	"github.com/mams/backend/internal/utils"
)

type Handler struct {
	services ServiceReader
	orgs     OrganizationReader
	releases ReleaseReader
	workflows WorkflowDispatcher
	kube     KubeDeployer
	callbackURL string
}

func NewHandler(services ServiceReader, orgs OrganizationReader, releases ReleaseReader, workflows WorkflowDispatcher, kube KubeDeployer, callbackURL string) *Handler {
	return &Handler{services: services, orgs: orgs, releases: releases, workflows: workflows, kube: kube, callbackURL: callbackURL}
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
			"author":         rel.Author,
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
	if err := h.applyKubeDeploy(r.Context(), svc, req.Environment, req.Strategy, req.Branch, req.GitTag); err != nil {
		log.Printf(
			"apply kube deploy failed: service_id=%s org_id=%s env=%s strategy=%s branch=%s git_tag=%s err=%v",
			svc.ID.String(), svc.OrganizationID.String(), req.Environment, req.Strategy, req.Branch, req.GitTag, err,
		)
		utils.WriteError(w, http.StatusInternalServerError, "failed to apply kubernetes rollout")
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
		"environment":  req.Environment,
		"strategy":     req.Strategy,
		"release_id":   created.ID.String(),
		"ref_name":     ref,
		"mams_callback_url": h.callbackURL + "/api/internal/releases/status",
	}
	if req.GitTag != "" {
		inputs["ref_type"] = "tag"
	} else {
		inputs["ref_type"] = "branch"
	}
	if req.GitTag != "" {
		inputs["git_tag"] = req.GitTag
	}

	if err := h.workflows.DispatchWorkflow(r.Context(), svc.RepositoryURL, "deploy.yml", ref, inputs); err != nil {
		log.Printf(
			"dispatch workflow failed: service_id=%s repo=%s workflow=%s ref=%s env=%s strategy=%s err=%v",
			svc.ID.String(), svc.RepositoryURL, "deploy.yml", ref, req.Environment, req.Strategy, err,
		)
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
	if err := h.applyKubeRollback(r.Context(), svc, req.Environment, req.GitTag); err != nil {
		log.Printf(
			"apply kube rollback failed: service_id=%s org_id=%s env=%s git_tag=%s err=%v",
			svc.ID.String(), svc.OrganizationID.String(), req.Environment, req.GitTag, err,
		)
		utils.WriteError(w, http.StatusInternalServerError, "failed to apply kubernetes rollback")
		return
	}

	created, err := h.CreatePending(r.Context(), svc.ID, claims.UserID, req.GitTag, "", req.Environment, "rollback", req.Description)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	inputs := map[string]string{
		"environment": req.Environment,
		"strategy":    "rollback",
		"release_id":  created.ID.String(),
		"ref_type":    "tag",
		"ref_name":    req.GitTag,
		"mams_callback_url": h.callbackURL + "/api/internal/releases/status",
		"git_tag":     req.GitTag,
		"rollback":    "true",
	}
	if err := h.workflows.DispatchWorkflow(r.Context(), svc.RepositoryURL, "deploy.yml", req.GitTag, inputs); err != nil {
		log.Printf(
			"dispatch rollback workflow failed: service_id=%s repo=%s workflow=%s ref=%s env=%s err=%v",
			svc.ID.String(), svc.RepositoryURL, "deploy.yml", req.GitTag, req.Environment, err,
		)
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

func (h *Handler) applyKubeDeploy(ctx context.Context, svc models.Service, env, strategy, branch, gitTag string) error {
	if h.kube == nil {
		return nil
	}
	slug := ""
	if h.orgs != nil {
		got, err := h.orgs.GetSlugByID(ctx, svc.OrganizationID)
		if err != nil {
			return err
		}
		slug = got
	}
	namespace := utils.BuildNamespace(slug, env)
	deployment := svc.Name
	container := "app"

	imageRef := gitTag
	if imageRef == "" {
		imageRef = branch
	}
	if imageRef == "" {
		imageRef = "latest"
	}

	registry := svc.ContainerRegistry
	if registry == "" {
		registry = "docker.io"
	}
	repo := extractRepoPath(svc.RepositoryURL)
	image := registry + "/" + repo + ":" + imageRef

	switch strategy {
	case "recreate":
		return h.kube.UpgradeRecreate(ctx, namespace, deployment, container, image)
	case "canary":
		return h.kube.ApplyCanaryPatch(ctx, namespace, deployment, deployment+"-canary", container, image, 1)
	default:
		return h.kube.UpgradeRolling(ctx, namespace, deployment, container, image)
	}
}

func (h *Handler) applyKubeRollback(ctx context.Context, svc models.Service, env, gitTag string) error {
	if h.kube == nil {
		return nil
	}
	slug := ""
	if h.orgs != nil {
		got, err := h.orgs.GetSlugByID(ctx, svc.OrganizationID)
		if err != nil {
			return err
		}
		slug = got
	}
	namespace := utils.BuildNamespace(slug, env)
	deployment := svc.Name
	container := "app"

	registry := svc.ContainerRegistry
	if registry == "" {
		registry = "docker.io"
	}
	repo := extractRepoPath(svc.RepositoryURL)
	image := registry + "/" + repo + ":" + gitTag

	return h.kube.RollbackToTag(ctx, namespace, deployment, container, image)
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

func extractRepoPath(repoURL string) string {
	repoURL = strings.TrimSuffix(repoURL, ".git")
	parts := strings.Split(repoURL, "/")
	if len(parts) >= 4 {
		owner := parts[len(parts)-2]
		repo := parts[len(parts)-1]
		return strings.ToLower(owner + "/" + repo)
	}
	if len(parts) >= 2 {
		return strings.ToLower(strings.Join(parts[len(parts)-2:], "/"))
	}
	return strings.ToLower(repoURL)
}
