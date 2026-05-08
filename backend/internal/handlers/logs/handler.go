package logs

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/mams/backend/internal/logx"
	authmw "github.com/mams/backend/internal/middleware/auth"
	"github.com/mams/backend/internal/models"
	"github.com/mams/backend/internal/utils"
	"github.com/mams/backend/internal/ws"
)

type logState struct {
	lastSnapshot []string
}

type Handler struct {
	logs          Reader
	k8sLogs       K8sLogReader
	services      ServiceGetter
	serviceLister ServiceLister
	orgs          OrgGetter
	wsHub         *ws.Hub
	log           *logx.Logger
	logStates     map[uuid.UUID]*logState
	statesMu      sync.RWMutex
	stopPolling   chan struct{}
	jwtValidator  JWTValidator
}

func NewHandler(logs Reader, k8sLogs K8sLogReader, services ServiceGetter, serviceLister ServiceLister, orgs OrgGetter, wsHub *ws.Hub, log *logx.Logger, jwtValidator JWTValidator) *Handler {
	h := &Handler{
		logs:          logs,
		k8sLogs:       k8sLogs,
		services:      services,
		serviceLister: serviceLister,
		orgs:          orgs,
		wsHub:         wsHub,
		log:           log,
		logStates:     make(map[uuid.UUID]*logState),
		stopPolling:   make(chan struct{}),
		jwtValidator:  jwtValidator,
	}
	go h.pollK8sLogs()
	return h
}

func (h *Handler) StopPolling() {
	close(h.stopPolling)
}

func (h *Handler) validateToken(ctx context.Context, token string) (authmw.Claims, error) {
	if h.jwtValidator == nil {
		return authmw.Claims{}, errors.New("jwt validator not configured")
	}
	return h.jwtValidator.Validate(token)
}

func (h *Handler) pollK8sLogs() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopPolling:
			log.Println("log polling stopped")
			return
		case <-ticker.C:
			h.checkK8sLogs()
		}
	}
}

func (h *Handler) checkK8sLogs() {
	if h.k8sLogs == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	services, err := h.fetchAllServices(ctx)
	if err != nil {
		h.log.ErrorCtx(ctx, "pollK8sLogs: fetch services failed", "err", err)
		return
	}

	for _, svc := range services {
		h.pollServiceLogs(ctx, svc)
	}
}

func (h *Handler) fetchAllServices(ctx context.Context) ([]models.Service, error) {
	if h.serviceLister == nil {
		return nil, errors.New("serviceLister not configured")
	}

	orgID := uuid.MustParse("300157c6-5599-4077-a3b2-922de677cf85")
	svcs, err := h.serviceLister.ListByOrganization(ctx, orgID)
	if err != nil {
		log.Printf("fetchAllServices: error=%v", err)
		return nil, err
	}
	log.Printf("fetchAllServices: found %d services", len(svcs))
	return svcs, nil
}

func (h *Handler) pollServiceLogs(ctx context.Context, svc models.Service) {
	slug, err := h.orgs.GetSlugByID(ctx, svc.OrganizationID)
	if err != nil {
		log.Printf("pollServiceLogs: GetSlugByID error=%v", err)
		return
	}

	namespace := utils.BuildNamespace(slug, "dev")

	rawLogs, err := h.k8sLogs.GetPodLogs(ctx, namespace, "app="+svc.Name, 50)
	if err != nil {
		log.Printf("pollServiceLogs: GetPodLogs error=%v namespace=%s", err, namespace)
		return
	}

	currentLines := normalizeLines(rawLogs)
	if len(currentLines) == 0 {
		return
	}

	h.statesMu.Lock()
	state, exists := h.logStates[svc.ID]
	if !exists {
		h.logStates[svc.ID] = &logState{
			lastSnapshot: append([]string(nil), currentLines...),
		}
		h.statesMu.Unlock()
		return
	}

	previousLines := append([]string(nil), state.lastSnapshot...)
	newLines := diffNewTail(previousLines, currentLines)
	state.lastSnapshot = append([]string(nil), currentLines...)
	h.statesMu.Unlock()

	if len(newLines) == 0 {
		return
	}

	for _, line := range newLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		level := detectLogLevel(line)

		saved := h.logs.Append(ctx, svc.ID, "dev", level, line)
		if saved == nil {
			log.Printf("pollServiceLogs: failed to append log service=%s message=%s", svc.Name, line)
			continue
		}

		log.Printf("BROADCAST: service=%s level=%s message=%s", svc.Name, level, line)
		h.wsHub.Broadcast(svc.ID, *saved)
	}
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

	if len(list) == 0 && h.k8sLogs != nil && filter.Level == "" && filter.Text == "" && filter.TimeFrom == nil && filter.TimeTo == nil {
		list = h.fetchFromK8s(r.Context(), serviceID)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Timestamp.After(list[j].Timestamp)
	})

	utils.WriteJSON(w, http.StatusOK, map[string]any{"logs": list})
}

func (h *Handler) fetchFromK8s(ctx context.Context, serviceID uuid.UUID) []models.LogEntry {
	svc, err := h.services.GetByID(ctx, serviceID)
	if err != nil {
		h.log.ErrorCtx(ctx, "fetchFromK8s: get service failed", "err", err)
		return nil
	}

	slug, err := h.orgs.GetSlugByID(ctx, svc.OrganizationID)
	if err != nil {
		h.log.ErrorCtx(ctx, "fetchFromK8s: get org failed", "err", err)
		return nil
	}

	env := "dev"
	namespace := utils.BuildNamespace(slug, env)

	rawLogs, err := h.k8sLogs.GetPodLogs(ctx, namespace, "app="+svc.Name, 200)
	if err != nil {
		h.log.ErrorCtx(ctx, "fetchFromK8s: get pod logs failed", "err", err, "namespace", namespace)
		return nil
	}

	lines := strings.Split(strings.TrimSpace(rawLogs), "\n")
	entries := make([]models.LogEntry, 0, len(lines))
	baseTime := time.Now()

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Делаем timestamp стабильнее внутри одного батча,
		// чтобы порядок не выглядел случайным.
		ts := baseTime.Add(-time.Duration(len(lines)-i) * time.Millisecond)

		entries = append(entries, models.LogEntry{
			ServiceID:   serviceID,
			Environment: env,
			Level:       "info",
			Message:     line,
			Timestamp:   ts,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	return entries
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

	log.Printf("WebSocket Stream: serviceID=%s", serviceID.String())

	token := r.URL.Query().Get("token")
	if token == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if token == "" {
		log.Printf("WebSocket Stream: no token in query or header")
		if _, ok := authmw.ClaimsFromContext(r.Context()); !ok {
			utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
	} else {
		log.Printf("WebSocket Stream: token present, validating")
		claims, err := h.validateToken(r.Context(), token)
		if err != nil {
			log.Printf("WebSocket Stream: token validation error=%v", err)
			utils.WriteError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		log.Printf("WebSocket Stream: token validated, orgID=%s", claims.OrganizationID)
		if claims.OrganizationID != uuid.Nil {
			ctx := authmw.WithClaims(r.Context(), claims)
			r = r.WithContext(ctx)
		}
	}

	claims, _ := authmw.ClaimsFromContext(r.Context())
	if claims.OrganizationID != uuid.Nil {
		svc, err := h.services.GetByID(r.Context(), serviceID)
		if err != nil || svc.OrganizationID != claims.OrganizationID {
			utils.WriteError(w, http.StatusNotFound, "service not found")
			return
		}
	}

	wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		CompressionMode:    websocket.CompressionContextTakeover,
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("WebSocket Accept error: %v", err)
		return
	}
	defer wsConn.Close(websocket.StatusNormalClosure, "")

	log.Printf("WebSocket connected: serviceID=%s", serviceID.String())

	conn := &ws.Conn{Send: make(chan []byte, 16)}
	h.wsHub.Join(serviceID, conn)
	defer h.wsHub.Leave(serviceID, conn)

	log.Printf("WebSocket joined hub: serviceID=%s", serviceID.String())

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

func normalizeLines(raw string) []string {
	parts := strings.Split(strings.TrimSpace(raw), "\n")
	result := make([]string, 0, len(parts))
	for _, line := range parts {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		result = append(result, line)
	}
	return result
}

func diffNewTail(prev, curr []string) []string {
	if len(curr) == 0 {
		return nil
	}

	if len(prev) == 0 {
		return nil
	}

	// Ищем самый длинный suffix prev, который является prefix curr.
	maxOverlap := 0
	maxPossible := min(len(prev), len(curr))

	for overlap := maxPossible; overlap > 0; overlap-- {
		if slicesEqual(prev[len(prev)-overlap:], curr[:overlap]) {
			maxOverlap = overlap
			break
		}
	}

	// Если overlap не найден, считаем что хвост полностью сменился
	// (например, рестарт pod'а). В этом случае лучше ничего не слать,
	// чем повторно заспамить весь хвост.
	if maxOverlap == 0 {
		return nil
	}

	if maxOverlap == len(curr) {
		return nil
	}

	return curr[maxOverlap:]
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func detectLogLevel(line string) string {
	lower := strings.ToLower(line)

	switch {
	case strings.Contains(lower, " level=debug "),
		strings.Contains(lower, "[debug]"),
		strings.Contains(lower, `"level":"debug"`),
		strings.Contains(lower, `"level":"dbg"`),
		strings.Contains(lower, " debug "):
		return "debug"

	case strings.Contains(lower, " level=warn "),
		strings.Contains(lower, " level=warning "),
		strings.Contains(lower, "[warn]"),
		strings.Contains(lower, "[warning]"),
		strings.Contains(lower, `"level":"warn"`),
		strings.Contains(lower, `"level":"warning"`),
		strings.Contains(lower, " warn "):
		return "warn"

	case strings.Contains(lower, " level=error "),
		strings.Contains(lower, "[error]"),
		strings.Contains(lower, `"level":"error"`),
		strings.Contains(lower, " error "):
		return "error"

	default:
		return "info"
	}
}
