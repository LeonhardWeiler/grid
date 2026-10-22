package internal

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStaticSecurityAndCaching(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "assets"), 0700)
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("canvas"), 0600)
	os.WriteFile(filepath.Join(dir, "assets", "index-hash.js"), []byte("test"), 0600)
	handler := SecurityHeaders(StaticHandler(dir))
	for _, test := range []struct {
		method, path string
		code         int
		cache        string
	}{
		{"GET", "/", 200, "no-cache"},
		{"GET", "/assets/index-hash.js", 200, "public, max-age=31536000, immutable"},
		{"GET", "/assets/", 404, ""},
		{"GET", "/missing", 404, ""},
		{"POST", "/", 405, ""},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(test.method, test.path, nil))
		if w.Code != test.code || w.Header().Get("Cache-Control") != test.cache || !strings.Contains(w.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") || w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("%s %s: %d %v", test.method, test.path, w.Code, w.Header())
		}
	}
}
