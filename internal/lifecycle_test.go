package internal

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestSlowClientQueueIsBoundedWithoutBlockingOtherClients(t *testing.T) {
	h := openTestHub(t, t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	slow := &Client{ctx: ctx, cancel: cancel, send: make(chan []byte, 32), ready: true}
	h.clients[nil] = slow
	for i := 0; i < 32; i++ {
		if !h.enqueue(slow, []byte("event")) {
			t.Fatal("queue filled too early")
		}
	}
	if h.enqueue(slow, []byte("overflow")) {
		t.Fatal("unbounded queue")
	}
	if len(h.clients) != 0 || ctx.Err() == nil || h.queueDrops.Load() != 1 {
		t.Fatal("slow client was not removed")
	}
	if slow.queued.Load() > maxQueuedBytes {
		t.Fatal("queue exceeded byte budget")
	}
	otherCtx, otherCancel := context.WithCancel(context.Background())
	defer otherCancel()
	other := &Client{ctx: otherCtx, cancel: otherCancel, send: make(chan []byte, 32)}
	if h.enqueue(other, make([]byte, maxQueuedBytes+1)) || other.queued.Load() != 0 {
		t.Fatal("oversized message accepted")
	}
}

func TestShutdownClosesWebSocketsAndPersistsAcknowledgedState(t *testing.T) {
	dir := t.TempDir()
	h := openTestHub(t, dir)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { HandleWS(h, w, r) }))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.CloseNow()
	if err := conn.Write(ctx, websocket.MessageText, []byte(`{"type":"init_client","clientId":"lifecycle","lastVersion":0}`)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := conn.Read(ctx); err != nil {
		t.Fatal(err)
	}
	response := handleSetPixel(h, "lifecycle", "", ClientMsg{X: 10, Y: 20, Color: "#ff0000"})
	if response.Type != "accepted" {
		t.Fatal(response)
	}
	done := make(chan error, 1)
	go func() { done <- h.Shutdown() }()
	for {
		if _, _, err := conn.Read(ctx); err != nil {
			if websocket.CloseStatus(err) < 0 {
				t.Fatalf("connection did not close cleanly: %v", err)
			}
			break
		}
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if h.Ready() {
		t.Fatal("stopping server stayed ready")
	}
	restored := openTestHub(t, dir)
	pixels, version := restored.store.Snapshot()
	if pixels["10:20"] != "#ff0000" || version != 1 {
		t.Fatal("shutdown lost an acknowledged pixel")
	}
}

func BenchmarkDenseSnapshot(b *testing.B) {
	pixels := make(map[string]string, GridSize*GridSize)
	for x := 0; x < GridSize; x++ {
		for y := 0; y < GridSize; y++ {
			pixels[fmt.Sprintf("%d:%d", x, y)] = "#ff0000"
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := encodeSnapshot(SnapshotFile{Schema: 1, Pixels: pixels, Version: len(pixels)}); err != nil {
			b.Fatal(err)
		}
	}
}
