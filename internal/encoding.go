package internal

import (
	"encoding/json"
	"strconv"
)

// Pixel coordinates and colors are already validated. JSON map ordering is not
// part of the protocol; avoid reflection, sorting and two allocations per cell.
func encodePixels(pixels map[string]string) json.RawMessage {
	b := make([]byte, 0, len(pixels)*20+2)
	b = append(b, '{')
	for key, color := range pixels {
		if len(b) > 1 {
			b = append(b, ',')
		}
		b = strconv.AppendQuote(b, key)
		b = append(b, ':')
		b = strconv.AppendQuote(b, color)
	}
	return append(b, '}')
}

func encodeSnapshot(state SnapshotFile) ([]byte, error) {
	metadata := state
	metadata.Pixels = nil
	b, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	// SnapshotFile omits nil pixels here; restore the required pixel object.
	b = b[:len(b)-1]
	b = append(b, `,"pixels":`...)
	b = append(b, encodePixels(state.Pixels)...)
	b = append(b, '}')
	return b, nil
}
