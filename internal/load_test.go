package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestSyncFloodDoesNotBlockLegitimateDrawing(t *testing.T) {
	h := openTestHub(t, t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { HandleWS(h, w, r) }))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	dial := func(id string) *websocket.Conn {
		t.Helper()
		c, _, err := websocket.Dial(ctx, server.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { c.CloseNow() })
		message, _ := json.Marshal(ClientMsg{Type: "init_client", ClientID: id})
		if err := c.Write(ctx, websocket.MessageText, message); err != nil {
			t.Fatal(err)
		}
		if _, _, err := c.Read(ctx); err != nil {
			t.Fatal(err)
		}
		return c
	}
	legitimate, abusive := dial("legitimate"), dial("abusive")
	started := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		close(started)
		for i := 0; i < 100; i++ {
			if abusive.Write(ctx, websocket.MessageText, []byte(`{"type":"sync","lastVersion":999999}`)) != nil {
				return
			}
		}
	}()
	<-started
	if err := legitimate.Write(ctx, websocket.MessageText, []byte(`{"type":"set_pixel","requestId":"normal-write","x":5,"y":6,"color":"#ff0000"}`)); err != nil {
		t.Fatal(err)
	}
	for {
		_, data, err := legitimate.Read(ctx)
		if err != nil {
			t.Fatal("legitimate drawing blocked by sync flood", err)
		}
		var response ServerResponse
		if err := json.Unmarshal(data, &response); err != nil {
			t.Fatal(err)
		}
		if response.Type == "accepted" {
			if response.RequestID != "normal-write" || response.Version != 1 {
				t.Fatal(response)
			}
			break
		}
	}
	for {
		if _, _, err := abusive.Read(ctx); err != nil {
			if websocket.CloseStatus(err) < 0 {
				t.Fatal("abusive client not closed by policy", err)
			}
			break
		}
	}
	<-done
}

func TestDenseStateCheckpointWriteLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("dense-state IO check")
	}
	h := openTestHub(t, t.TempDir())
	pixels := make(map[string]string, GridSize*GridSize)
	for x := 0; x < GridSize; x++ {
		for y := 0; y < GridSize; y++ {
			pixels[fmt.Sprintf("%d:%d", x, y)] = "#ff0000"
		}
	}
	h.store.LoadSnapshot(pixels, len(pixels))
	var background sync.WaitGroup
	background.Go(func() {
		for i := 0; i < 3; i++ {
			if err := h.Save(); err != nil {
				t.Error(err)
			}
		}
	})
	background.Go(func() {
		for i := 0; i < 3; i++ {
			h.stateMu.Lock()
			h.snapshotLocked()
			h.stateMu.Unlock()
		}
	})
	latencies := make([]time.Duration, 80)
	for i := range latencies {
		started := time.Now()
		response := handleSetPixel(h, fmt.Sprintf("load-%d", i), "", ClientMsg{X: i, Y: 0, Color: "#0000ff"})
		latencies[i] = time.Since(started)
		if response.Type != "accepted" {
			t.Fatal(response)
		}
	}
	background.Wait()
	slices.Sort(latencies)
	t.Logf("full canvas + concurrent init/checkpoint: n=%d p50=%v p95=%v p99=%v", len(latencies), latencies[40], latencies[76], latencies[79])
	if latencies[79] > 3*time.Second {
		t.Fatal("write latency exceeds the isolated test's 3s safety budget")
	}
}
