package internal

import (
	"testing"
)

func TestCheckpointRetainsWritesNewerThanCapturedSnapshot(t *testing.T) {
	dir := t.TempDir()
	h := openTestHub(t, dir)
	handleSetPixel(h, "alice", "", ClientMsg{X: 1, Color: "#ff0000"})
	pixels, version := h.store.Snapshot()
	handleSetPixel(h, "bob", "", ClientMsg{X: 2, Color: "#00ff00"})
	if err := h.persistence.Save(pixels, version, nil, nil); err != nil {
		t.Fatal(err)
	}
	restored := openTestHub(t, dir)
	pixels, version = restored.store.Snapshot()
	if version != 2 || len(pixels) != 2 {
		t.Fatal("compaction lost concurrent write")
	}
}

func TestUnchangedSaveDoesNotRewriteSnapshot(t *testing.T) {
	h := openTestHub(t, t.TempDir())
	if err := h.Save(); err != nil {
		t.Fatal(err)
	}
	saved := h.lastSave.Load()
	if err := h.Save(); err != nil {
		t.Fatal(err)
	}
	if h.lastSave.Load() != saved {
		t.Fatal("unchanged checkpoint rewritten")
	}
}
