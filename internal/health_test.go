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

func TestCheckpointFailureRecoversAcknowledgedJournalAfterDiskRepair(t *testing.T) {
	dir := t.TempDir()
	h := openTestHub(t, dir)
	if response := handleSetPixel(h, "before-failure", "", ClientMsg{X: 3, Y: 4, Color: "#ff0000"}); response.Type != "accepted" {
		t.Fatal(response)
	}
	// An unusable snapshot destination simulates a durable checkpoint failure,
	// while the acknowledged WAL remains readable and must not be destroyed.
	path := filepath.Join(dir, "snapshot.json")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	err := h.Save()
	if err == nil {
		t.Fatal("checkpoint failure was hidden")
	}
	h.markFailed(err)
	if h.Ready() {
		t.Fatal("failed checkpoint stayed ready")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	recovered := openTestHub(t, dir)
	if recovered.store.Version() != 1 {
		t.Fatal("lost durable write after repair")
	}
	if response := handleSetPixel(recovered, "after-repair", "", ClientMsg{X: 4, Y: 4, Color: "#0000ff"}); response.Type != "accepted" || response.Version != 2 {
		t.Fatal("writing did not recover", response)
	}
}
