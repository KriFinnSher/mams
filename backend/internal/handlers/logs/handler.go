package logs

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/mams/backend/internal/logx"
	authmw "github.com/mams/backend/internal/middleware/auth"
	"github.com/mams/backend/internal/models"
	"github.com/mams/backend/internal/utils"
	"github.com/mams/backend/internal/ws"
)

type Handler struct {
	logs  Reader
	wsHub *ws.Hub
	log   *logx.Logger
}

func NewHandler(logs Reader, wsHub *ws.Hub, log *logx.Logger) *Handler {
	return &Handler{logs: logs, wsHub: wsHub, log: log}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	serviceID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}
	filter, err := parseLogFilter(r)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	list, err := h.logs.ListByService(r.Context(), serviceID, filter)
	if err != nil {
		h.log.ErrorCtx(r.Context(), "list logs failed", "err", err, "service_id", serviceID)
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	utils.WriteJSON(w, http.StatusOK, map[string]any{"logs": list})
}

func parseLogFilter(r *http.Request) (models.LogFilter, error) {
	filter := models.LogFilter{
		Level: r.URL.Query().Get("level"),
		Text:  r.URL.Query().Get("text"),
		Limit: 200,
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n <= 0 {
			return models.LogFilter{}, errors.New("invalid limit")
		}
		filter.Limit = n
	}
	if v := r.URL.Query().Get("time_from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return models.LogFilter{}, errors.New("invalid time_from")
		}
		filter.TimeFrom = &t
	}
	if v := r.URL.Query().Get("time_to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return models.LogFilter{}, errors.New("invalid time_to")
		}
		filter.TimeTo = &t
	}
	return filter, nil
}

func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	serviceID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}
	if _, ok := authmw.ClaimsFromContext(r.Context()); !ok {
		utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{CompressionMode: websocket.CompressionContextTakeover})
	if err != nil {
		return
	}
	defer wsConn.Close(websocket.StatusNormalClosure, "")

	conn := &ws.Conn{Send: make(chan []byte, 16)}
	h.wsHub.Join(serviceID, conn)
	defer h.wsHub.Leave(serviceID, conn)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-conn.Send:
			if !ok {
				return
			}
			if err := wsConn.Write(ctx, websocket.MessageText, data); err != nil {
				return
			}
		}
	}
}

func (h *Handler) Ingest(w http.ResponseWriter, r *http.Request) {
	serviceID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}
	var entries []struct {
		Environment string `json:"environment"`
		Level       string `json:"level"`
		Message     string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&entries); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	for _, entry := range entries {
		logEntry := h.logs.Append(r.Context(), serviceID, entry.Environment, entry.Level, entry.Message)
		if logEntry != nil {
			h.wsHub.Broadcast(serviceID, *logEntry)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}
