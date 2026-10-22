package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotValidation(t *testing.T) {
	for _, content := range []string{`null`, `{}`, `{"pixels":null,"version":0}`, `{"pixels":{},"version":-1}`, `{"pixels":{"999:999":"#ff0000"},"version":1}`, `{"pixels":{"0:0":"bad"},"version":1}`, `{"pixels":{"00:0":"#ff0000"},"version":1}`, `{"pixels":{},"version":0,"schema":2}`} {
		t.Run(content, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "snapshot.json"), []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := OpenHub(dir); err == nil {
				t.Fatal("accepted invalid snapshot")
			}
		})
	}
	for _, content := range []string{`{"pixels":{},"version":0}`, `{"pixels":{"0:0":"#ff0000"},"version":1}`} {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "snapshot.json"), []byte(content), 0600)
		if _, err := OpenHub(dir); err != nil {
			t.Fatal(err)
		}
	}
}
