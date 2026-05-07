package services

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	authmw "github.com/mams/backend/internal/middleware/auth"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
	"github.com/mams/backend/internal/utils"
)

func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
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
		h.log.ErrorCtx(r.Context(), "get service before metrics failed", "err", err, "service_id", id)
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if svc.OrganizationID != claims.OrganizationID {
		utils.WriteError(w, http.StatusNotFound, "service not found")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{
		"service_id":            svc.ID.String(),
		"grafana_dashboard_uid": svc.GrafanaDashboardUID,
		"embed_url":             buildGrafanaEmbedURL(h.cfg.GrafanaURL, svc.GrafanaDashboardUID),
	})
}

func buildGrafanaEmbedURL(baseURL, uid string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	uid = strings.TrimSpace(uid)
	if baseURL == "" || uid == "" {
		return ""
	}

	return baseURL + "/d/" + uid + "?kiosk"
}
