package internal

import (
	"fmt"
	"net/http/httptest"
	"testing"
)

func TestSourceBudgetSurvivesIdentityChangesAndRestart(t *testing.T) {
	dir := t.TempDir()
	h := openTestHub(t, dir)
	r := httptest.NewRequest("GET", "/ws", nil)
	r.RemoteAddr = "192.0.2.1:1234"
	source := sourceKey(r)
	for i := 0; i < 10; i++ {
		handleSetPixel(h, fmt.Sprintf("client-%d", i), source, ClientMsg{X: i, Color: "#ff0000"})
	}
	if h.store.Version() != sourceBurst {
		t.Fatalf("identity changes bypassed budget: %d", h.store.Version())
	}
	restored := openTestHub(t, dir)
	handleSetPixel(restored, "after-restart", source, ClientMsg{Color: "#ff0000"})
	if restored.store.Version() != sourceBurst {
		t.Fatal("restart bypassed source budget")
	}
}

func TestProxyTrust(t *testing.T) {
	r := httptest.NewRequest("GET", "/ws", nil)
	r.RemoteAddr = "192.0.2.1:1234"
	direct := sourceKey(r)
	r.Header.Set("X-Real-IP", "198.51.100.1")
	t.Setenv("TRUST_PROXY", "1")
	if sourceKey(r) != direct {
		t.Fatal("trusted header from remote client")
	}
	r.RemoteAddr = "127.0.0.1:1234"
	first := sourceKey(r)
	r.Header.Set("X-Real-IP", "198.51.100.2")
	if sourceKey(r) == first {
		t.Fatal("ignored trusted proxy")
	}
}
