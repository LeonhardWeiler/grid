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
