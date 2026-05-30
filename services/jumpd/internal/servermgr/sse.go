package servermgr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sting8k/jump/services/jumpd/internal/jwtauth"
)

// SSEHub manages SSE connections and broadcasts events to connected clients.
type SSEHub struct {
	mu      sync.RWMutex
	clients map[string][]chan SSEEvent // userID -> channels
}

// SSEEvent represents a server-sent event.
type SSEEvent struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// NewSSEHub creates a new SSE hub.
func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients: make(map[string][]chan SSEEvent),
	}
}

// Subscribe adds a client channel for a user and returns it.
func (h *SSEHub) Subscribe(userID string) chan SSEEvent {
	ch := make(chan SSEEvent, 64)
	h.mu.Lock()
	h.clients[userID] = append(h.clients[userID], ch)
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes a client channel for a user.
func (h *SSEHub) Unsubscribe(userID string, ch chan SSEEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	channels := h.clients[userID]
	for i, c := range channels {
		if c == ch {
			h.clients[userID] = append(channels[:i], channels[i+1:]...)
			close(ch)
			break
		}
	}
	if len(h.clients[userID]) == 0 {
		delete(h.clients, userID)
	}
}

// Broadcast sends an event to all clients of a specific user.
func (h *SSEHub) Broadcast(userID string, event SSEEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.clients[userID] {
		select {
		case ch <- event:
		default:
			// Drop if buffer full — client is too slow.
		}
	}
}

// BroadcastAll sends an event to all connected clients.
func (h *SSEHub) BroadcastAll(event SSEEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, channels := range h.clients {
		for _, ch := range channels {
			select {
			case ch <- event:
			default:
			}
		}
	}
}

// Handler returns the HTTP handler for the SSE endpoint.
func (h *SSEHub) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := jwtauth.UserIDFromContext(r.Context())
		if userID == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing user")
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		ch := h.Subscribe(userID)
		defer h.Unsubscribe(userID, ch)

		// Send initial connected event.
		writeSSE(w, "connected", map[string]string{"status": "ok"})
		flusher.Flush()

		heartbeat := time.NewTicker(30 * time.Second)
		defer heartbeat.Stop()

		notify := r.Context().Done()
		for {
			select {
			case <-notify:
				return
			case <-heartbeat.C:
				fmt.Fprint(w, ":\n\n")
				flusher.Flush()
			case ev, open := <-ch:
				if !open {
					return
				}
				writeSSE(w, ev.Type, ev.Data)
				flusher.Flush()
			}
		}
	}
}

func writeSSE(w http.ResponseWriter, event string, data any) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, bytes)
}
