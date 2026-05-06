package services

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	authmw "github.com/mams/backend/internal/middleware/auth"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
	"github.com/mams/backend/internal/utils"
)

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
		h.log.Error("get service by id failed", "err", err, "service_id", id)
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if svc.OrganizationID != claims.OrganizationID {
		utils.WriteError(w, http.StatusNotFound, "service not found")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{
		"id":                            svc.ID.String(),
		"organization_id":               svc.OrganizationID.String(),
		"created_by_user_id":            svc.CreatedByUserID.String(),
		"owner_user_id":                 svc.OwnerUserID.String(),
		"name":                          svc.Name,
		"description":                   svc.Description,
		"type":                          svc.Type,
		"version":                       svc.Version,
		"test_coverage":                 svc.TestCoverage,
		"minimum_test_coverage_enabled": svc.MinimumTestCoverageEnabled,
		"minimum_test_coverage":         svc.MinimumTestCoverage,
		"pii_sensitive":                 svc.PIISensitive,
		"responsible_team_ref":          svc.ResponsibleTeamRef,
		"importance":                    svc.Importance,
		"repository_url":                svc.RepositoryURL,
		"default_branch":                svc.DefaultBranch,
		"grafana_dashboard_uid":         svc.GrafanaDashboardUID,
	})
}
