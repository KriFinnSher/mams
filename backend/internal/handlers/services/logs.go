package services

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mams/backend/internal/models"
	"github.com/mams/backend/internal/utils"
)

func (h *Handler) GetLogs(w http.ResponseWriter, r *http.Request) {
	serviceID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}

	filter := models.LogFilter{
		Level: r.URL.Query().Get("level"),
		Text:  r.URL.Query().Get("text"),
		Limit: 200,
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n <= 0 {
			utils.WriteError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		filter.Limit = n
	}
	if v := r.URL.Query().Get("time_from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			utils.WriteError(w, http.StatusBadRequest, "invalid time_from")
			return
		}
		filter.TimeFrom = &t
	}
	if v := r.URL.Query().Get("time_to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			utils.WriteError(w, http.StatusBadRequest, "invalid time_to")
			return
		}
		filter.TimeTo = &t
	}

	logs, err := h.logs.ListByService(r.Context(), serviceID, filter)
	if err != nil {
		h.log.ErrorCtx(r.Context(), "list logs failed", "err", err, "service_id", serviceID)
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	utils.WriteJSON(w, http.StatusOK, map[string]any{"logs": logs})
}

