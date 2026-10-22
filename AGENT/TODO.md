# 🧱 grid — TODO

---

# 🔴 P0 - CRITICAL (Stability / Data loss prevention)

## [X] Persistence System (DONE)

* Save `PixelStore` to disk (JSON snapshot)
* Load snapshot on startup
* Autosave loop (5–30s interval)
* Version restored correctly after restart

---

## [X] Event Log Limitation (DONE)

* Ring buffer implemented (`MaxEvents`)
* `GetSince()` supports version-based delta sync
* No unbounded memory growth anymore

---

## [⚠️] Hub concurrency hardening (PARTIAL / REVIEW REQUIRED)

### Status: mostly correct, but still needs audit

### MUST VERIFY:

#### ✔ Properly locked:

* `connToID` → OK (mutex used)
* `idToClients` → OK
* `lastAction` → OK
* `clients` → OK

#### ⚠️ STILL CHECK:

* iteration over maps under lock (all safe?)
* any external access to `Store` consistency with hub state
* `RemoveClient()` edge timing (cancel vs send close)

### Action:

* full concurrency audit pass (no logic changes, only verification)

---

# 🟠 P1 - IMPORTANT (scaling + abuse resistance)

## [X] Improved rate limiting (DONE)

* cooldown system per client ID
* optional IP fallback conceptually supported
* basic burst protection via timestamp checks

---

## [X] Broadcast stability (DONE - MVP level)

* non-blocking send via `select { default: }`
* slow clients do not block system
* auto-removal of dead clients

---

## [X] Server-side validation hardening (DONE)

* coordinate validation enforced
* color validation enforced
* message type filtering implemented
* invalid message counter protection exists

---

# 🟡 P2 - CLIENT RELIABILITY (HIGH IMPACT UX)

## [X] Client reconnect handling (DONE)

* reconnect logic exists
* version stored in localStorage
* backend supports `GetSince(lastVersion)`

---

## [⚠️] Client state machine (IMPROVEMENT NEEDED)

### Status: PARTIAL (needs final correctness pass)

### REQUIRED STATES:

```ts
connecting
handshaking
syncing
connected
reconnecting
disconnected
```

### RULE (IMPORTANT FIX):

* ❌ DO NOT set `connected` in `onopen`
* ✔ ONLY set `connected` after:

  * `init`
  * OR successful `sync`

---

## [X] Basic sync strategy (DONE)

* delta sync via events (`GetSince`)
* fallback to full snapshot if needed
* version tracking works correctly

---

## [⚠️] Sync optimization (OPTIONAL IMPROVEMENT)

* chunking not required yet
* current system is fine for MVP scale

---

# 🟢 P3 - PERFORMANCE / SCALING (FUTURE)

## [ ] Remove JSON overhead (FUTURE)

* binary protocol (WebSocket framing optimization)
* reduce allocations in hot path

---

## [ ] Chunked canvas system (FUTURE SCALE STEP)

* divide grid into regions
* sync only dirty chunks
* reduce bandwidth per client

---

## [ ] Per-client write queue (ADVANCED)

* isolate slow clients fully
* optional backpressure system

---

# 🔵 P4 - FEATURES (PRODUCT EXPANSION)

## [X] Minimap UI (DONE)

---

## [ ] Pixel history (who painted what)

* store user attribution per pixel
* optional audit trail system

---

## [ ] Heatmap system

* track pixel activity frequency
* render intensity overlay

---

## [ ] Mobile gesture support

* pinch zoom
* pan canvas
* touch painting

---

## [ ] Timelapse / replay system

* replay full canvas evolution from event log

---

## [ ] Factions / teams system

* optional multiplayer grouping layer
* territory mechanics (future gameplay layer)

---

# 🧭 FINAL REALITY CHECK (IMPORTANT)

## ✔ Already solid:

* WebSocket backend architecture
* PixelStore + versioning system
* cooldown system
* persistence
* delta sync
* basic reconnect support

## ⚠️ Only real remaining “core gap”:

👉 **Frontend connection state correctness (state machine refinement)**

---

# 🚀 NEXT BEST STEP (clear recommendation)

If you want maximum stability improvement with minimal work:

### 👉 DO NEXT:

1. Fix frontend state machine (remove fake "connected")
2. Add proper reconnect backoff + jitter
3. Ensure init/sync sets connected state
