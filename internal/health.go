package internal

import (
	"fmt"
	"net/http"
)

// Readiness never waits for a persistence lock or a slow disk operation.
func (h *Hub) Ready() bool            { return !h.failed.Load() && !h.closing.Load() }
func (h *Hub) Failed() bool           { return h.failed.Load() }
func (h *Hub) Failures() <-chan error { return h.failures }
func (h *Hub) markFailed(err error) {
	h.failed.Store(true)
	select {
	case h.failures <- err:
	default:
	}
}
func (h *Hub) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if !h.Ready() {
		http.Error(w, "storage unavailable or server stopping", http.StatusServiceUnavailable)
		return
	}
	fmt.Fprintln(w, "ok")
}
