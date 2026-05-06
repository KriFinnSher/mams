package services

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/mams/backend/internal/middleware/auth"
	"github.com/mams/backend/internal/utils"
	"github.com/mams/backend/internal/ws"
	"nhooyr.io/websocket"
)

func (h *Handler) StreamLogs(w http.ResponseWriter, r *http.Request) {
	serviceID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}

	if _, ok := auth.ClaimsFromContext(r.Context()); !ok {
		utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		CompressionMode: websocket.CompressionContextTakeover,
	})
	if err != nil {
		return
	}
	defer c.Close(websocket.StatusNormalClosure, "")

	conn := &ws.Conn{
		Send: make(chan []byte, 16),
	}
	h.wsHub.Join(serviceID, conn)
	defer h.wsHub.Leave(serviceID, conn)

	ctx := r.Context()

	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		c.Close(websocket.StatusNormalClosure, "")
		close(done)
	}()

	for {
		select {
		case <-done:
			return
		case data, ok := <-conn.Send:
			if !ok {
				return
			}
			if err := c.Write(ctx, websocket.MessageText, data); err != nil {
				return
			}
		}
	}
}

func (h *Handler) IngestLog(w http.ResponseWriter, r *http.Request) {
	serviceID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}

	if r.Method != http.MethodPost {
		utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var entry struct {
		Environment string `json:"environment"`
		Level       string `json:"level"`
		Message     string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	logEntry := h.logs.Append(r.Context(), serviceID, entry.Environment, entry.Level, entry.Message)
	if logEntry != nil {
		h.wsHub.Broadcast(serviceID, *logEntry)
	}

	w.WriteHeader(http.StatusNoContent)
}