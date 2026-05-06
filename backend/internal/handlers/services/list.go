package services

import (
	"net/http"

	"github.com/mams/backend/internal/logx"
	authmw "github.com/mams/backend/internal/middleware/auth"
	"github.com/mams/backend/internal/utils"
)

type Handler struct {
	services ServiceReader
	log      *logx.Logger
}

func NewHandler(services ServiceReader, log *logx.Logger) *Handler {
	return &Handler{services: services, log: log}
}

type serviceItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := authmw.ClaimsFromContext(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	list, err := h.services.ListByOrganization(r.Context(), claims.OrganizationID)
	if err != nil {
		h.log.ErrorCtx(r.Context(), "list services by organization failed", "err", err)
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := make([]serviceItem, 0, len(list))
	for _, s := range list {
		resp = append(resp, serviceItem{
			ID:   s.ID.String(),
			Name: s.Name,
		})
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"services": resp})
}
