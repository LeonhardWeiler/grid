package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// The JSON journal belongs to one process. The kernel releases this lock on
// crashes as well as normal exit, without stale PID-file recovery rules.
func LockStateDir(dir string) (*os.File, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(filepath.Join(dir, ".lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		return nil, fmt.Errorf("state directory is already in use: %w", err)
	}
	return file, nil
}
