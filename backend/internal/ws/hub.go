package ws

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
	"github.com/mams/backend/internal/models"
)

type Hub struct {
	mu       sync.RWMutex
	services map[uuid.UUID]map[*Conn]struct{}
}

type Conn struct {
	Send chan []byte
}

func NewHub() *Hub {
	return &Hub{
		services: make(map[uuid.UUID]map[*Conn]struct{}),
	}
}

func (h *Hub) Join(serviceID uuid.UUID, c *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.services[serviceID] == nil {
		h.services[serviceID] = make(map[*Conn]struct{})
	}
	h.services[serviceID][c] = struct{}{}
}

func (h *Hub) Leave(serviceID uuid.UUID, c *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conns, ok := h.services[serviceID]; ok {
		delete(conns, c)
		if len(conns) == 0 {
			delete(h.services, serviceID)
		}
	}
}

func (h *Hub) Broadcast(serviceID uuid.UUID, entry models.LogEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	if conns, ok := h.services[serviceID]; ok {
		for c := range conns {
			select {
			case c.Send <- data:
			default:
			}
		}
	}
}