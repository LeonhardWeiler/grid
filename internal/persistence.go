package internal

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type SnapshotFile struct {
	Schema        int               `json:"schema,omitempty"`
	Pixels        map[string]string `json:"pixels,omitempty"`
	Version       int               `json:"version"`
	LastAction    map[string]int64  `json:"lastAction,omitempty"`
	SourceActions map[string]int64  `json:"sourceActions,omitempty"`
}

type JournalEntry struct {
	Event
	ClientID   string `json:"clientId"`
	Time       int64  `json:"time"`
	Source     string `json:"source,omitempty"`
	SourceTime int64  `json:"sourceTime,omitempty"`
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

func (p *Persistence) Save(pixels map[string]string, version int, lastAction, sources map[string]int64) error {
	b, err := encodeSnapshot(SnapshotFile{Schema: 1, Pixels: pixels, Version: version, LastAction: lastAction, SourceActions: sources})
	if err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	// Newer writes may have arrived while encoding. Retain them during
	// compaction; the old WAL remains intact until the snapshot is durable.
	var retained []byte
	journal, err := os.Open(p.filePath + ".wal")
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(journal)
	for scanner.Scan() {
		var entry JournalEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			journal.Close()
			return fmt.Errorf("cannot compact corrupt journal: %w", err)
		}
		if entry.Version > version {
			retained = append(retained, scanner.Bytes()...)
			retained = append(retained, '\n')
		}
	}
	err = scanner.Err()
	journal.Close()
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
	wal, err := os.CreateTemp(filepath.Dir(p.filePath), ".journal-*")
	if err != nil {
		return err
	}
	defer os.Remove(wal.Name())
	if _, err = wal.Write(retained); err != nil {
		wal.Close()
		return err
	}
	if err = wal.Sync(); err != nil {
		wal.Close()
		return err
	}
	if err = wal.Close(); err != nil {
		return err
	}
	if err = os.Rename(wal.Name(), p.filePath+".wal"); err != nil {
		return err
	}
	return dir.Sync()
}

func (p *Persistence) Load() (SnapshotFile, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	state := SnapshotFile{Pixels: make(map[string]string), LastAction: make(map[string]int64)}
	b, err := os.ReadFile(p.filePath)
	if err == nil {
		var fields map[string]json.RawMessage
		if err = json.Unmarshal(b, &fields); err != nil || fields == nil || fields["pixels"] == nil || fields["version"] == nil || string(fields["pixels"]) == "null" || string(fields["version"]) == "null" {
			return state, fmt.Errorf("snapshot requires pixels and version")
		}
		if err = json.Unmarshal(b, &state); err != nil {
			return state, err
		}
		if err = validateSnapshot(state); err != nil {
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
	if state.SourceActions == nil {
		state.SourceActions = make(map[string]int64)
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
			if entry.Source != "" {
				if !validSource(entry.Source) || entry.SourceTime < entry.Time {
					return state, fmt.Errorf("invalid journal source")
				}
				state.SourceActions[entry.Source] = entry.SourceTime
			}
		}
		if readErr != nil {
			break
		}
	}
	return state, nil
}

func validateSnapshot(s SnapshotFile) error {
	if s.Schema < 0 || s.Schema > 1 || s.Version < 0 || len(s.Pixels) > GridSize*GridSize || len(s.Pixels) > s.Version || len(s.LastAction) > 10000 {
		return fmt.Errorf("invalid snapshot metadata")
	}
	for key, color := range s.Pixels {
		xs, ys, ok := strings.Cut(key, ":")
		x, xe := strconv.Atoi(xs)
		y, ye := strconv.Atoi(ys)
		if !ok || xe != nil || ye != nil || !ValidCoords(x, y) || fmt.Sprintf("%d:%d", x, y) != key || !ValidColor(color) {
			return fmt.Errorf("invalid snapshot pixel %q", key)
		}
	}
	for id, timestamp := range s.LastAction {
		if !ValidClientID(id) || timestamp < 0 {
			return fmt.Errorf("invalid snapshot cooldown")
		}
	}
	if len(s.SourceActions) > 10000 {
		return fmt.Errorf("too many source budgets")
	}
	for source, timestamp := range s.SourceActions {
		if !validSource(source) || timestamp < 0 {
			return fmt.Errorf("invalid source budget")
		}
	}
	return nil
}

func validSource(source string) bool {
	_, err := hex.DecodeString(source)
	return len(source) == 64 && err == nil
}
