package services

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	authmw "github.com/mams/backend/internal/middleware/auth"
	"github.com/mams/backend/internal/models"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
	"github.com/mams/backend/internal/utils"
)

type updateInfoRequest struct {
	Description        string `json:"description"`
	Type               string `json:"type"`
	TestCoverage       int    `json:"test_coverage"`
	PIISensitive       bool   `json:"pii_sensitive"`
	ResponsibleTeamRef string `json:"responsible_team_ref"`
	Importance         string `json:"importance"`
	RepositoryURL      string `json:"repository_url"`
	DefaultBranch      string `json:"default_branch"`
	GrafanaDashboardUID string `json:"grafana_dashboard_uid"`
}

func (h *Handler) UpdateInfo(w http.ResponseWriter, r *http.Request) {
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

	current, err := h.services.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, postgresrepo.ErrServiceNotFound) {
			utils.WriteError(w, http.StatusNotFound, "service not found")
			return
		}
		h.log.ErrorCtx(r.Context(), "get service before update failed", "err", err, "service_id", id)
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if current.OrganizationID != claims.OrganizationID {
		utils.WriteError(w, http.StatusNotFound, "service not found")
		return
	}

	var req updateInfoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validateUpdateInfo(req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := h.services.UpdateInfo(r.Context(), models.Service{
		ID:                 id,
		Description:        req.Description,
		Type:               req.Type,
		TestCoverage:       req.TestCoverage,
		PIISensitive:       req.PIISensitive,
		ResponsibleTeamRef: req.ResponsibleTeamRef,
		Importance:         req.Importance,
		RepositoryURL:      req.RepositoryURL,
		DefaultBranch:      req.DefaultBranch,
		GrafanaDashboardUID: req.GrafanaDashboardUID,
	})
	if err != nil {
		if errors.Is(err, postgresrepo.ErrServiceNotFound) {
			utils.WriteError(w, http.StatusNotFound, "service not found")
			return
		}
		h.log.ErrorCtx(r.Context(), "update service info failed", "err", err, "service_id", id)
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{
		"id":                    updated.ID.String(),
		"description":           updated.Description,
		"type":                  updated.Type,
		"test_coverage":         updated.TestCoverage,
		"pii_sensitive":         updated.PIISensitive,
		"responsible_team_ref":  updated.ResponsibleTeamRef,
		"importance":            updated.Importance,
		"repository_url":        updated.RepositoryURL,
		"default_branch":        updated.DefaultBranch,
		"grafana_dashboard_uid": updated.GrafanaDashboardUID,
	})
}

