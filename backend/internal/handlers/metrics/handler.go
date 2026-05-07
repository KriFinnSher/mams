package metrics

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	authmw "github.com/mams/backend/internal/middleware/auth"
	"github.com/mams/backend/internal/models"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
	"github.com/mams/backend/internal/utils"
)

type ServiceReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (models.Service, error)
}

type Handler struct {
	services   ServiceReader
	grafanaURL string
}

func NewHandler(services ServiceReader, grafanaURL string) *Handler {
	return &Handler{services: services, grafanaURL: grafanaURL}
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
	utils.WriteJSON(w, http.StatusOK, map[string]any{
		"service_id":            svc.ID.String(),
		"grafana_dashboard_uid": svc.GrafanaDashboardUID,
		"embed_url":             buildGrafanaEmbedURL(h.grafanaURL, svc.GrafanaDashboardUID),
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

