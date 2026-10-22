package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime"
	"time"

	"github.com/coder/websocket"
)

var Revision = "dev"

func Version(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]any{"revision": Revision, "go": runtime.Version(), "protocol": Protocol.Protocol, "gridSize": GridSize})
}

// Backups contain the public drawing and version, never browser/source IDs.
func (h *Hub) Backup(w http.ResponseWriter, r *http.Request) {
	if !h.Ready() {
		http.Error(w, "server unavailable", 503)
		return
	}
	h.stateMu.Lock()
	pixels, version := h.store.Snapshot()
	h.stateMu.Unlock()
	b, err := encodeSnapshot(SnapshotFile{Schema: 1, Pixels: pixels, Version: version})
	if err != nil {
		http.Error(w, "snapshot failed", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Disposition", `attachment; filename="snapshot.json"`)
	w.Write(b)
}

func (h *Hub) Metrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.Header().Set("Cache-Control", "no-store")
	h.mu.RLock()
	clients := len(h.clients)
	h.mu.RUnlock()
	ready := 0
	if h.Ready() {
		ready = 1
	}
	fmt.Fprintf(w, "grid_ready %d\ngrid_clients %d\ngrid_version %d\ngrid_checkpoint_version %d\ngrid_last_checkpoint_timestamp_seconds %d\ngrid_accepted_total %d\ngrid_rejected_total %d\ngrid_queue_drops_total %d\ngrid_connections_rejected_total %d\ngrid_sync_total %d\n", ready, clients, h.store.Version(), h.checkpointVersion.Load(), h.lastSave.Load()/1000, h.accepted.Load(), h.rejected.Load(), h.queueDrops.Load(), h.connectionRejects.Load(), h.syncs.Load())
}

func (h *Hub) AdminHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.Health)
	mux.HandleFunc("GET /metrics", h.Metrics)
	mux.HandleFunc("GET /backup", h.Backup)
	return mux
}

// Probe performs an independent read-only WebSocket initialization, not just
// a TCP upgrade or a process-liveness check. It is also used by the NAS monitor.
func Probe(ctx context.Context, endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("probe requires http(s)")
	}
	u.Path = "/readyz"
	u.RawQuery = ""
	u.Fragment = ""
	request, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	response.Body.Close()
	if response.StatusCode != 200 {
		return fmt.Errorf("readiness returned %d", response.StatusCode)
	}
	if u.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}
	u.Path = "/ws"
	conn, _, err := websocket.Dial(ctx, u.String(), nil)
	if err != nil {
		return err
	}
	defer conn.CloseNow()
	conn.SetReadLimit(16 << 20)
	b, _ := json.Marshal(ClientMsg{Type: "init_client", ClientID: "monitor", LastVersion: 0})
	if err := conn.Write(ctx, websocket.MessageText, b); err != nil {
		return err
	}
	_, data, err := conn.Read(ctx)
	if err != nil {
		return err
	}
	var initial struct {
		Type    string            `json:"type"`
		Pixels  map[string]string `json:"pixels"`
		Version int               `json:"version"`
	}
	if err := json.Unmarshal(data, &initial); err != nil {
		return err
	}
	if initial.Type != "init" {
		return fmt.Errorf("expected authoritative init")
	}
	if err := validateSnapshot(SnapshotFile{Pixels: initial.Pixels, Version: initial.Version}); err != nil {
		return err
	}
	return nil
}

func CheckState(dir string) error {
	state, err := NewPersistence(filepath.Join(dir, "snapshot.json")).Load()
	if err != nil {
		return err
	}
	fmt.Printf("valid version=%d pixels=%d\n", state.Version, len(state.Pixels))
	return nil
}

func (h *Hub) RunStatusLog(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.mu.RLock()
			clients := len(h.clients)
			h.mu.RUnlock()
			log.Printf("state clients=%d version=%d checkpoint=%d accepted=%d rejected=%d queue_drops=%d ready=%t", clients, h.store.Version(), h.checkpointVersion.Load(), h.accepted.Load(), h.rejected.Load(), h.queueDrops.Load(), h.Ready())
		}
	}
}
