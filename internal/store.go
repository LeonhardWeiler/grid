package internal

import "sync"

type Event struct {
	Version int    `json:"version"`
	X       int    `json:"x"`
	Y       int    `json:"y"`
	Color   string `json:"color"`
}

type PixelStore struct {
	mu      sync.RWMutex
	pixels  map[string]string
	events  []Event
	start   int
	count   int
	version int
}

func NewPixelStore() *PixelStore {
	return &PixelStore{
		pixels: make(map[string]string),
		events: make([]Event, MaxEvents),
	}
}

func (s *PixelStore) Version() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}

func (s *PixelStore) Set(key string, x, y int, color string) Event {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.version++

	s.pixels[key] = color

	ev := Event{
		Version: s.version,
		X:       x,
		Y:       y,
		Color:   color,
	}

	var idx int

	if s.count == MaxEvents {
		idx = s.start
		s.start = (s.start + 1) % MaxEvents
	} else {
		idx = (s.start + s.count) % MaxEvents
		s.count++
	}

	s.events[idx] = ev

	return ev
}

func (s *PixelStore) GetSince(v int) ([]Event, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if v < 0 || v > s.version {
		return nil, false
	}
	if s.count == 0 {
		return []Event{}, v == s.version
	}

	oldest := s.events[s.start].Version

	if v < oldest-1 {
		return nil, false
	}

	res := make([]Event, 0, s.count)

	for i := 0; i < s.count; i++ {
		idx := (s.start + i) % MaxEvents
		e := s.events[idx]

		if e.Version > v {
			res = append(res, e)
		}
	}

	return res, true
}

func (s *PixelStore) Snapshot() (map[string]string, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	copy := make(map[string]string, len(s.pixels))

	for k, v := range s.pixels {
		copy[k] = v
	}

	version := s.version

	return copy, version
}

func (s *PixelStore) LoadSnapshot(
	pixels map[string]string,
	version int,
) {
	s.mu.Lock()
	defer s.mu.Unlock()

	copy := make(map[string]string, len(pixels))
	for k, v := range pixels {
		copy[k] = v
	}
	s.pixels = copy
	s.version = version
}
