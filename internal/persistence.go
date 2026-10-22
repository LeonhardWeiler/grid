package internal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type SnapshotFile struct {
	Pixels     map[string]string `json:"pixels"`
	Version    int               `json:"version"`
	LastAction map[string]int64  `json:"lastAction,omitempty"`
}

type JournalEntry struct {
	Event
	ClientID string `json:"clientId"`
	Time     int64  `json:"time"`
}

type Persistence struct {
	filePath string
	mu       sync.Mutex
}

func NewPersistence(filePath string) *Persistence { return &Persistence{filePath: filePath} }

// Append is durable before the corresponding pixel is broadcast to any client.
func (p *Persistence) Append(entry JournalEntry) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	b, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(p.filePath+".wal", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = f.Write(append(b, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

func (p *Persistence) Save(pixels map[string]string, version int, lastAction map[string]int64) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	b, err := json.Marshal(SnapshotFile{Pixels: pixels, Version: version, LastAction: lastAction})
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(p.filePath), ".snapshot-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(f.Name(), p.filePath); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(p.filePath))
	if err != nil {
		return err
	}
	defer dir.Close()
	if err = dir.Sync(); err != nil {
		return err
	}
	// Only compact the journal after the replacement snapshot is durable.
	wal, err := os.OpenFile(p.filePath+".wal", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer wal.Close()
	return wal.Sync()
}

func (p *Persistence) Load() (SnapshotFile, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	state := SnapshotFile{Pixels: make(map[string]string), LastAction: make(map[string]int64)}
	b, err := os.ReadFile(p.filePath)
	if err == nil {
		if err = json.Unmarshal(b, &state); err != nil {
			return state, err
		}
	} else if !os.IsNotExist(err) {
		return state, err
	}
	if state.Pixels == nil {
		state.Pixels = make(map[string]string)
	}
	if state.LastAction == nil {
		state.LastAction = make(map[string]int64)
	}
	f, err := os.OpenFile(p.filePath+".wal", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return state, err
	}
	defer f.Close()
	// Persist the journal's directory entry before acknowledging the first pixel.
	if err := f.Sync(); err != nil {
		return state, err
	}
	dir, err := os.Open(filepath.Dir(p.filePath))
	if err != nil {
		return state, err
	}
	err = dir.Sync()
	dir.Close()
	if err != nil {
		return state, err
	}
	reader := bufio.NewReader(f)
	var offset int64
	for {
		line, readErr := reader.ReadBytes('\n')
		if readErr != nil && readErr != io.EOF {
			return state, readErr
		}
		if len(line) == 0 {
			break
		}
		if line[len(line)-1] != '\n' {
			// A crash may leave an unacknowledged partial final write. Remove it
			// before appending again; never silently skip complete corrupt records.
			if err := f.Truncate(offset); err != nil {
				return state, err
			}
			if err := f.Sync(); err != nil {
				return state, err
			}
			break
		}
		var entry JournalEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			return state, fmt.Errorf("journal at %d: %w", offset, err)
		}
		offset += int64(len(line))
		if entry.Version > state.Version {
			if entry.Version != state.Version+1 || !ValidCoords(entry.X, entry.Y) || !ValidColor(entry.Color) || !ValidClientID(entry.ClientID) {
				return state, fmt.Errorf("invalid journal entry at %d", offset)
			}
			state.Pixels[fmt.Sprintf("%d:%d", entry.X, entry.Y)] = entry.Color
			state.Version = entry.Version
			state.LastAction[entry.ClientID] = entry.Time
		}
		if readErr != nil {
			break
		}
	}
	return state, nil
}
