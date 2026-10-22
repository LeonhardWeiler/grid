package internal

import "testing"

func TestStateDirectoryHasOneOwner(t *testing.T) {
	dir := t.TempDir()
	first, err := LockStateDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	if second, err := LockStateDir(dir); err == nil {
		second.Close()
		t.Fatal("two processes could own the journal")
	}
	first.Close()
	next, err := LockStateDir(dir)
	if err != nil {
		t.Fatal("lock was not released", err)
	}
	next.Close()
}
