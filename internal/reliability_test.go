package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func openTestHub(t *testing.T, dir string) *Hub {
	t.Helper()
	h, err := OpenHub(dir)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestJournalSurvivesCrashAndCompaction(t *testing.T) {
	dir := t.TempDir()
	h := openTestHub(t, dir)
	handleSetPixel(h, "alice", ClientMsg{X: 7, Y: 9, Color: "#ff0000"})
	// Reopen without calling Save or Shutdown, simulating a killed process.
	restored := openTestHub(t, dir)
	pixels, version := restored.Store.Snapshot()
	if pixels["7:9"] != "#ff0000" || version != 1 || restored.GetCooldownEnd("alice") <= time.Now().UnixMilli() {
		t.Fatalf("lost state: %v, %d", pixels, version)
	}
	handleSetPixel(restored, "alice", ClientMsg{X: 8, Y: 9, Color: "#00ff00"})
	if restored.Store.Version() != 1 {
		t.Fatal("restart bypassed cooldown")
	}
	if err := restored.Save(); err != nil {
		t.Fatal(err)
	}
	wal, err := os.ReadFile(filepath.Join(dir, "snapshot.json.wal"))
	if err != nil || len(wal) != 0 {
		t.Fatalf("journal not compacted: %v", err)
	}
	handleSetPixel(restored, "bob", ClientMsg{X: 8, Y: 9, Color: "#00ff00"})
	again := openTestHub(t, dir)
	pixels, version = again.Store.Snapshot()
	if len(pixels) != 2 || version != 2 {
		t.Fatalf("snapshot + journal recovery: %v, %d", pixels, version)
	}
}

func TestJournalTruncationAndCorruption(t *testing.T) {
	dir := t.TempDir()
	h := openTestHub(t, dir)
	handleSetPixel(h, "alice", ClientMsg{Color: "#ff0000"})
	path := filepath.Join(dir, "snapshot.json.wal")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(`{"version":2`)
	f.Close()
	restored := openTestHub(t, dir)
	handleSetPixel(restored, "bob", ClientMsg{X: 1, Color: "#00ff00"})
	if openTestHub(t, dir).Store.Version() != 2 {
		t.Fatal("partial tail poisoned future writes")
	}
	if err := os.WriteFile(path, []byte("corrupt\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenHub(dir); err == nil {
		t.Fatal("silently discarded corrupt journal")
	}
}

func TestConcurrentSaveAndWrites(t *testing.T) {
	dir := t.TempDir()
	h := openTestHub(t, dir)
	var wg sync.WaitGroup
	for i := range 30 {
		wg.Go(func() {
			handleSetPixel(h, fmt.Sprintf("client-%d", i), ClientMsg{X: i, Y: i, Color: "#ff0000"})
			if err := h.Save(); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	restored := openTestHub(t, dir)
	pixels, version := restored.Store.Snapshot()
	if len(pixels) != 30 || version != 30 {
		t.Fatalf("concurrent persistence lost state: %d pixels at %d", len(pixels), version)
	}
}

func TestSnapshotFallbackAfterRestart(t *testing.T) {
	s := NewPixelStore()
	s.LoadSnapshot(map[string]string{"1:2": "#ff0000"}, 100)
	s.Set("2:3", 2, 3, "#00ff00")
	if _, ok := s.GetSince(1); ok {
		t.Fatal("delta sync over missing history")
	}
	if _, ok := s.GetSince(102); ok {
		t.Fatal("accepted future version")
	}
	if events, ok := s.GetSince(100); !ok || len(events) != 1 {
		t.Fatal("valid adjacent delta missing")
	}
}

func TestWebSocketReloadSnapshotAndBroadcast(t *testing.T) {
	h := openTestHub(t, t.TempDir())
	handleSetPixel(h, "prior", ClientMsg{X: 1, Y: 2, Color: "#ff0000"})
	handleSetPixel(h, "other", ClientMsg{X: 2, Y: 3, Color: "#00ff00"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { HandleWS(h, w, r) }))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dial := func(id string) *websocket.Conn {
		t.Helper()
		conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.CloseNow() })
		b, _ := json.Marshal(ClientMsg{Type: "init_client", ClientID: id, LastVersion: 1})
		if err := conn.Write(ctx, websocket.MessageText, b); err != nil {
			t.Fatal(err)
		}
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var initial ServerResponse
		if err := json.Unmarshal(data, &initial); err != nil {
			t.Fatal(err)
		}
		if initial.Type != "init" || len(initial.Pixels) != 2 || initial.Version != 2 {
			t.Fatalf("reload received incomplete canvas: %s", data)
		}
		return conn
	}
	first := dial("alice")
	second := dial("bob")
	if err := first.Write(ctx, websocket.MessageText, []byte(`{"type":"set_pixel","x":3,"y":4,"color":"#0000ff"}`)); err != nil {
		t.Fatal(err)
	}
	for _, conn := range []*websocket.Conn{first, second} {
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var event struct {
			Event
			Type string `json:"type"`
		}
		if err := json.Unmarshal(data, &event); err != nil {
			t.Fatal(err)
		}
		if event.Type != "pixel_update" || event.Version != 3 || event.Color != "#0000ff" {
			t.Fatalf("bad broadcast: %s", data)
		}
	}
	readCooldown := func() {
		t.Helper()
		_, data, err := first.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var response ServerResponse
		if err := json.Unmarshal(data, &response); err != nil {
			t.Fatal(err)
		}
		if response.Type != "cooldown" || response.CooldownMs <= 0 || response.CooldownMs > CooldownMs {
			t.Fatalf("missing server-reported remaining duration: %s", data)
		}
	}
	readCooldown()
	// A blocked attempt must return the remaining duration, so a client whose
	// display expired too early can recover without waiting for a heartbeat.
	if err := first.Write(ctx, websocket.MessageText, []byte(`{"type":"set_pixel","x":4,"y":4,"color":"#0000ff"}`)); err != nil {
		t.Fatal(err)
	}
	readCooldown()
	if h.Store.Version() != 3 {
		t.Fatal("cooldown allowed an extra pixel")
	}
	if err := first.Write(ctx, websocket.MessageText, []byte(`{"type":"sync","lastVersion":3}`)); err != nil {
		t.Fatal(err)
	}
	_, data, err := first.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var synced ServerResponse
	if err := json.Unmarshal(data, &synced); err != nil {
		t.Fatal(err)
	}
	if synced.Type != "sync" || synced.CooldownMs <= 0 || synced.CooldownMs > CooldownMs {
		t.Fatalf("sync lost remaining cooldown: %s", data)
	}

}

func TestReloadWaitsForQuietAndIsBounded(t *testing.T) {
	h := openTestHub(t, t.TempDir())
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() { h.WaitForQuiet(ctx, 100*time.Millisecond, 600*time.Millisecond); close(done) }()
	time.Sleep(180 * time.Millisecond)
	handleSetPixel(h, "alice", ClientMsg{Color: "#ff0000"})
	select {
	case <-done:
		t.Fatal("reload kicked clients before quiet window")
	default:
	}
	<-done
	if time.Since(started) < 280*time.Millisecond {
		t.Fatal("reload ignored last drawing")
	}
	h.stateMu.Lock()
	h.lastWrite = time.Now().Add(time.Hour)
	h.stateMu.Unlock()
	started = time.Now()
	h.WaitForQuiet(ctx, time.Hour, 30*time.Millisecond)
	if time.Since(started) > 200*time.Millisecond {
		t.Fatal("busy canvas blocked reload indefinitely")
	}
}
