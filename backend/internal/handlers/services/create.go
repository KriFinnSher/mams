package services

import (
	"encoding/json"
	"net/http"

	"github.com/mams/backend/internal/models"
	authmw "github.com/mams/backend/internal/middleware/auth"
	"github.com/mams/backend/internal/utils"
)

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := authmw.ClaimsFromContext(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := req.validate(); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	created, err := h.services.Create(r.Context(), models.Service{
		OrganizationID:             claims.OrganizationID,
		CreatedByUserID:            claims.UserID,
		OwnerUserID:                claims.UserID,
		Name:                       req.Name,
		Description:                req.Description,
		Type:                       req.Type,
		Version:                    "",
		TestCoverage:               req.TestCoverage,
		MinimumTestCoverageEnabled: req.MinimumTestCoverageEnabled,
		MinimumTestCoverage:        req.MinimumTestCoverage,
		PIISensitive:               req.PIISensitive,
		ResponsibleTeamRef:         req.ResponsibleTeamRef,
		Importance:                 req.Importance,
		RepositoryURL:              req.RepositoryURL,
		DefaultBranch:              req.DefaultBranch,
		GrafanaDashboardUID:        req.GrafanaDashboardUID,
	})
	if err != nil {
		h.log.ErrorCtx(r.Context(), "create service failed", "err", err, "name", req.Name)
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, map[string]string{"id": created.ID.String()})
}
