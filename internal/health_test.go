package internal

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFailureDisablesReadinessAndRetainsJournal(t *testing.T) {
	dir := t.TempDir()
	h := openTestHub(t, dir)
	path := filepath.Join(dir, "snapshot.json.wal")
	os.Remove(path)
	os.Mkdir(path, 0700)
	handleSetPixel(h, "client", "", ClientMsg{Color: "#ff0000"})
	r := httptest.NewRecorder()
	h.Health(r, httptest.NewRequest("GET", "/healthz", nil))
	if r.Code != 503 || h.Ready() {
		t.Fatal("storage failure reported healthy")
	}
	select {
	case <-h.Failures():
	default:
		t.Fatal("no restart notification")
	}
	if err := h.Save(); err == nil {
		t.Fatal("compacted a failed journal")
	}
	if h.store.Version() != 0 {
		t.Fatal("acknowledged a failed write")
	}
}
