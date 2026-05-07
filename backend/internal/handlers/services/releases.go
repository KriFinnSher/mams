package services

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	authmw "github.com/mams/backend/internal/middleware/auth"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
	"github.com/mams/backend/internal/utils"
)

func (h *Handler) GetReleases(w http.ResponseWriter, r *http.Request) {
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
		h.log.ErrorCtx(r.Context(), "get service before releases failed", "err", err, "service_id", id)
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if svc.OrganizationID != claims.OrganizationID {
		utils.WriteError(w, http.StatusNotFound, "service not found")
		return
	}

	list, err := h.releases.ListByService(r.Context(), id)
	if err != nil {
		h.log.ErrorCtx(r.Context(), "list releases failed", "err", err, "service_id", id)
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
