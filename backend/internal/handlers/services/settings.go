package services

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	authmw "github.com/mams/backend/internal/middleware/auth"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
	"github.com/mams/backend/internal/utils"
)

type updateSettingsRequest struct {
	Settings map[string]any `json:"settings"`
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
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
		h.log.ErrorCtx(r.Context(), "get service before update settings failed", "err", err, "service_id", id)
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if current.OrganizationID != claims.OrganizationID {
		utils.WriteError(w, http.StatusNotFound, "service not found")
		return
	}

	var req updateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Settings == nil {
		req.Settings = map[string]any{}
	}
	if err := validateSettingsPatch(req.Settings); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if touchesOwnerOnlySettings(req.Settings) && current.OwnerUserID != claims.UserID {
		utils.WriteError(w, http.StatusForbidden, "owner permissions required")
		return
	}

	updated, err := h.services.UpdateSettings(r.Context(), id, req.Settings)
	if err != nil {
		if errors.Is(err, postgresrepo.ErrServiceNotFound) {
			utils.WriteError(w, http.StatusNotFound, "service not found")
			return
		}
		h.log.ErrorCtx(r.Context(), "update service settings failed", "err", err, "service_id", id)
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{
		"id":       updated.ID.String(),
		"settings": updated.Settings,
	})
}

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
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
		h.log.ErrorCtx(r.Context(), "get service settings failed", "err", err, "service_id", id)
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if current.OrganizationID != claims.OrganizationID {
		utils.WriteError(w, http.StatusNotFound, "service not found")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{
		"id":       current.ID.String(),
		"settings": current.Settings,
	})
}
