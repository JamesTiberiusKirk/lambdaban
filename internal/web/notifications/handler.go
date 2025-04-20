package notifications

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/JamesTiberiusKirk/lambdaban/internal/metrics"
	"github.com/alexedwards/scs/v2"
)

// Notification is the message sent to the client
type Notification struct {
	Type    string
	Content string
}

// SSEConnection holds the notification channel and a done signal
type SSEConnection struct {
	NotifyCh chan Notification
	Done     chan struct{}
}

// NotificationsHandler manages all SSE connections
type NotificationsHandler struct {
	log     *slog.Logger
	m       *metrics.Metrics
	mu      sync.RWMutex
	clients map[string]*SSEConnection // userID -> connection
	sm      *scs.SessionManager
}

// NewNotificationsHandler creates a new handler
func NewNotificationsHandler(log *slog.Logger, m *metrics.Metrics, sm *scs.SessionManager) *NotificationsHandler {
	return &NotificationsHandler{
		log:     log,
		m:       m,
		clients: make(map[string]*SSEConnection),
		sm:      sm,
	}
}

type event struct {
	Data  string `json:"data"`
	Event string `json:"event"`
}

// ServeHTTP upgrades the connection to SSE and manages notifications
func (h *NotificationsHandler) ServeSSE(w http.ResponseWriter, r *http.Request) {
	userID := h.sm.GetString(r.Context(), "user")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		h.log.Error("SSE Unauthorized")
		return
	}

	// Ensure only one connection per user
	h.mu.Lock()
	if oldConn, ok := h.clients[userID]; ok {
		close(oldConn.Done)
		delete(h.clients, userID)
	}
	conn := &SSEConnection{
		NotifyCh: make(chan Notification, 8),
		Done:     make(chan struct{}),
	}
	h.clients[userID] = conn
	h.mu.Unlock()

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	h.m.SSENotificationConnections.Add(1)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		h.log.Error("Streaming unsupported")
		return
	}

	// Cleanup on disconnect
	defer func() {
		h.mu.Lock()
		delete(h.clients, userID)
		h.mu.Unlock()
		close(conn.NotifyCh)
		h.m.SSENotificationConnections.Sub(1)
	}()

	ctx := r.Context()

	for {
		select {
		case <-ctx.Done():
			return
		case <-conn.Done:
			return
		case n := <-conn.NotifyCh:
			var buf bytes.Buffer
			err := notification(n).Render(r.Context(), &buf)
			if err != nil {
				fmt.Fprintf(w, "event: error\ndata: %s\n\n", "error creating notification")
				continue
			}

			fmt.Fprintf(w, "event: notification\ndata: %s\n\n", buf.String())
			flusher.Flush()
		}
	}
}

// Notify sends a notification to the user's SSE connection
func (h *NotificationsHandler) Notify(userID string, n Notification) {
	h.mu.RLock()
	conn, ok := h.clients[userID]
	h.mu.RUnlock()
	if !ok {
		return // No active connection
	}
	select {
	case conn.NotifyCh <- n:
	default:
		// Channel full, drop or handle overflow
	}
}
