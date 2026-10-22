package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/http"
	"net/netip"
	"os"
	"time"
)

const sourceBurst = 3
const maxClients = 256
const maxSourceClients = 12
const maxQueuedBytes = 8 << 20

type admission struct {
	debt   int64
	active int
	seen   int64
}

func (h *Hub) admit(source string, now int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for key, a := range h.admissions {
		if a.active == 0 && now-a.seen > 60000 {
			delete(h.admissions, key)
		}
	}
	a := h.admissions[source]
	if a == nil {
		if len(h.admissions) >= 1024 {
			return false
		}
		a = &admission{}
		h.admissions[source] = a
	}
	if h.openConnections >= maxClients || a.active >= maxSourceClients {
		return false
	}
	next, retry := advanceBudget(a.debt, now, 1000, 3)
	global, globalRetry := advanceBudget(h.handshakeBudget, now, 250, 4)
	if retry > 0 || globalRetry > 0 {
		return false
	}
	a.debt = next
	a.seen = now
	a.active++
	h.openConnections++
	h.handshakeBudget = global
	return true
}

func (h *Hub) releaseAdmission(source string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.openConnections--
	h.admissions[source].active--
}

// Trust only nginx on loopback, with an explicitly enabled, overwritten header.
func sourceKey(r *http.Request) string {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return "invalid"
	}
	if addr.IsLoopback() && os.Getenv("TRUST_PROXY") == "1" {
		if forwarded, err := netip.ParseAddr(r.Header.Get("X-Real-IP")); err == nil {
			addr = forwarded
		}
	}
	addr = addr.Unmap()
	if addr.Is6() {
		addr = netip.PrefixFrom(addr, 64).Masked().Addr()
	}
	hash := sha256.Sum256([]byte(addr.String()))
	return hex.EncodeToString(hash[:])
}

// A persisted leaky bucket allows a small shared-network burst, while changing
// IDs cannot raise its sustained rate. No raw IP address is stored.
func advanceBudget(previous, now, interval int64, burst int) (next, retry int64) {
	next = max(previous, now) + interval
	if next > now+interval*int64(burst) {
		return previous, next - now - interval*int64(burst)
	}
	return next, 0
}

func (h *Hub) sourceCooldown(source string, now int64) (next, retry int64) {
	if source == "" {
		return 0, 0
	}
	return advanceBudget(h.sourceActions[source], now, CooldownMs, sourceBurst)
}

func (h *Hub) cleanupSources() {
	now := time.Now().UnixMilli()
	for source, debt := range h.sourceActions {
		if debt <= now {
			delete(h.sourceActions, source)
		}
	}
}
