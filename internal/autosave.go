package internal

import (
	"context"
	"fmt"
	"log"
	"maps"
	"time"

	"github.com/coder/websocket"
)

func (h *Hub) Save() error {
	h.saveMu.Lock()
	defer h.saveMu.Unlock()
	h.stateMu.Lock()
	if h.failed.Load() {
		h.stateMu.Unlock()
		return fmt.Errorf("storage failed; journal retained for recovery")
	}
	h.mu.RLock()
	oldActions := len(h.lastAction)
	h.mu.RUnlock()
	oldSources := len(h.sourceActions)
	h.CleanupLastAction(CooldownMs)
	h.cleanupSources()
	h.mu.RLock()
	sameMetadata := oldActions == len(h.lastAction) && oldSources == len(h.sourceActions)
	h.mu.RUnlock()
	if h.savedVersion == h.store.Version() && sameMetadata {
		h.stateMu.Unlock()
		return nil
	}
	snap, version := h.store.Snapshot()
	h.mu.RLock()
	actions := maps.Clone(h.lastAction)
	h.mu.RUnlock()
	sources := maps.Clone(h.sourceActions)
	h.stateMu.Unlock()
	watchdog := time.AfterFunc(5*time.Second, func() { h.markFailed(fmt.Errorf("checkpoint exceeded 5 seconds")) })
	defer watchdog.Stop()
	if err := h.persistence.Save(snap, version, actions, sources); err != nil {
		return err
	}
	h.savedVersion = version
	h.checkpointVersion.Store(int64(version))
	h.lastSave.Store(time.Now().UnixMilli())
	return nil
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
				h.markFailed(err)
				return
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
	h.closing.Store(true)
	h.stateMu.Lock()
	h.stopping = true
	h.stateMu.Unlock()
	var err error
	if !h.failed.Load() {
		err = h.Save()
		if err != nil {
			h.markFailed(err)
		}
	}
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
