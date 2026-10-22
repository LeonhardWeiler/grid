package internal

import "testing"

func TestDeltaRingBoundaries(t *testing.T) {
	s := NewPixelStore()
	for i := 0; i < MaxEvents+3; i++ {
		s.Set("0:0", 0, 0, "#ff0000")
	}
	for _, tc := range []struct {
		version, count int
		ok             bool
	}{
		{-1, 0, false}, {2, 0, false}, {3, MaxEvents, true},
		{MaxEvents + 2, 1, true}, {MaxEvents + 3, 0, true}, {MaxEvents + 4, 0, false},
	} {
		events, ok := s.GetSince(tc.version)
		if ok != tc.ok || len(events) != tc.count {
			t.Fatalf("version %d: %d events, ok=%v", tc.version, len(events), ok)
		}
		for i, event := range events {
			if event.Version != tc.version+i+1 {
				t.Fatal("noncontiguous delta")
			}
		}
	}
	if allocations := testing.AllocsPerRun(20, func() { s.GetSince(s.Version()) }); allocations != 0 {
		t.Fatalf("empty delta allocated %v times", allocations)
	}
}

func BenchmarkCurrentDelta(b *testing.B) {
	s := NewPixelStore()
	for i := 0; i < MaxEvents; i++ {
		s.Set("0:0", 0, 0, "#ff0000")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.GetSince(MaxEvents)
	}
}
