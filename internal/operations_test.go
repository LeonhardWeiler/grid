package internal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBackupExcludesIdentitiesAndRestoresDrawing(t *testing.T) {
	h := openTestHub(t, t.TempDir())
	handleSetPixel(h, "private-browser", "", ClientMsg{X: 1, Y: 2, Color: "#ff0000"})
	w := httptest.NewRecorder()
	h.Backup(w, httptest.NewRequest("GET", "/backup", nil))
	if w.Code != 200 || strings.Contains(w.Body.String(), "private-browser") {
		t.Fatal("backup leaked identity")
	}
	var state SnapshotFile
	if err := json.Unmarshal(w.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	if state.Version != 1 || state.Pixels["1:2"] != "#ff0000" || len(state.LastAction) != 0 || len(state.SourceActions) != 0 {
		t.Fatal("invalid backup")
	}
}

func TestReadOnlyProbeAndMetrics(t *testing.T) {
	h := openTestHub(t, t.TempDir())
	mux := http.NewServeMux()
	mux.HandleFunc("/readyz", h.Health)
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) { HandleWS(h, w, r) })
	server := httptest.NewServer(mux)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := Probe(ctx, server.URL); err != nil {
		t.Fatal(err)
	}
	if h.store.Version() != 0 {
		t.Fatal("probe changed drawing")
	}
	w := httptest.NewRecorder()
	h.Metrics(w, httptest.NewRequest("GET", "/metrics", nil))
	if !strings.Contains(w.Body.String(), "grid_ready 1\n") {
		t.Fatal("missing readiness metric")
	}
}
