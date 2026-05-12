package notifications

import (
	"encoding/json"
	"sync"
)

type Client interface {
	Send(payload []byte) bool
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[Client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[Client]struct{})}
}

func (h *Hub) Subscribe(userID string, client Client) func() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[userID] == nil {
		h.clients[userID] = make(map[Client]struct{})
	}
	h.clients[userID][client] = struct{}{}

	return func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		delete(h.clients[userID], client)
		if len(h.clients[userID]) == 0 {
			delete(h.clients, userID)
		}
	}
}

func (h *Hub) Publish(userID string, event RealtimeEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}

	h.mu.RLock()
	clients := make([]Client, 0, len(h.clients[userID]))
	for client := range h.clients[userID] {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		if !client.Send(payload) {
			h.remove(userID, client)
		}
	}
}

func (h *Hub) remove(userID string, client Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.clients[userID], client)
	if len(h.clients[userID]) == 0 {
		delete(h.clients, userID)
	}
}
