package internal

import (
	"context"
	"log"
	"time"

	"github.com/coder/websocket"
)

func (h *Hub) Save() error {
	h.stateMu.Lock()
	defer h.stateMu.Unlock()
	return h.saveLocked()
}

func (h *Hub) saveLocked() error {
	h.CleanupLastAction(CooldownMs)
	snap, version := h.Store.Snapshot()
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.Persistence.Save(snap, version, h.lastAction)
}

func (h *Hub) RunAutosave(ctx context.Context) {
	ticker := time.NewTicker(AutosaveInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := h.Save(); err != nil {
				log.Printf("autosave failed: %v", err)
			}
		}
	}
}

// Reload leaves existing clients connected and able to draw until a quiet
// window, with a bounded wait so busy canvases still receive updates.
func (h *Hub) WaitForQuiet(ctx context.Context, quiet, maximum time.Duration) {
	started := time.Now()
	deadline := time.NewTimer(maximum)
	defer deadline.Stop()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-deadline.C:
			return
		case <-ticker.C:
			h.stateMu.Lock()
			idle := time.Since(h.lastWrite) >= quiet && time.Since(started) >= quiet
			h.stateMu.Unlock()
			if idle {
				return
			}
		}
	}
}

func (h *Hub) Shutdown() error {
	h.stateMu.Lock()
	h.stopping = true
	err := h.saveLocked()
	h.stateMu.Unlock()
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for _, client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()
	for _, client := range clients {
		client.cancel()
		go client.conn.Close(websocket.StatusServiceRestart, "server restarting; reconnect")
	}
	return err
}
