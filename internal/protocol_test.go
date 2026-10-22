package internal

import "testing"

func TestRequiredMessageFields(t *testing.T) {
	for _, data := range []string{`null`, `{}`, `{"type":"other"}`, `{"type":"set_pixel","color":"#ff0000"}`, `{"type":"set_pixel","x":null,"y":0,"color":"#ff0000"}`, `{"type":"sync"}`} {
		if _, err := decodeClientMsg([]byte(data)); err == nil {
			t.Fatalf("accepted %s", data)
		}
	}
	if msg, err := decodeClientMsg([]byte(`{"type":"set_pixel","x":0,"y":0,"color":"#ff0000"}`)); err != nil || msg.X != 0 || msg.Y != 0 {
		t.Fatal("valid origin rejected", err)
	}
}
