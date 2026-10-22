# 🟦 grid — Updated Roadmap (Current State Aligned)

---

# PHASE 1 — Correctness & Connection Lifecycle (HIGH PRIORITY)

## 1. Fix hub concurrency edge cases (SendToClientID + iteration safety)

### Goal

Eliminate hidden race conditions + safe map iteration under lock.

### Status

⚠️ Partially OK, but still fragile under load.

### Required improvements

* Ensure **all early returns release RLock**
* Avoid holding references to maps after unlock
* Snapshot `idToClients` safely before iteration

### Result

Fully race-safe hub under concurrent broadcasts.

---

## 2. Implement proper connection state machine (FRONTEND)

### Goal

Make connection UI reflect REAL sync state, not socket state.

### Replace current implicit logic with:

```ts
connecting
handshaking
syncing
connected
reconnecting
disconnected
```

### Important fix

❌ `connected` must NOT be set in `onopen`

✔ Only after:

* `init`
* or `sync` success

### Result

No fake-ready UI states anymore.

---

## 3. Only mark “connected” after full sync

### Goal

Prevent UI showing ready canvas before pixel data exists.

### Move logic:

From:

```ts
ws.onopen → connected
```

To:

```ts
on init OR sync message → connected
```

### Result

Deterministic readiness state.

---

## 4. Reconnect system upgrade (already partially done)

### Current state

✔ reconnect loop exists
❌ still not fully stabilized

### Required improvements

* exponential backoff
* jitter (±300–800ms)
* reset delay on successful sync

### Result

No reconnect storms under outage.

---

## 5. UI connection overlay refinement

### Goal

Accurate user feedback during sync lifecycle

### States:

* Connecting socket...
* Handshaking...
* Syncing canvas...
* Reconnecting...
* Offline

### Result

Debuggable and user-friendly connection UX

---

# PHASE 2 — Persistence Hardening (CRITICAL FOR DATA SAFETY)

## 6. Add autosave loop (server-side)

### Goal

Prevent data loss without heavy write overhead

### Status

⚠️ Already partially implemented conceptually

### Required:

* periodic snapshot save (10–30s interval)
* version included

### Result

Crash-safe canvas state recovery

---

## 7. Add graceful shutdown persistence

### Goal

Avoid last-second data loss

### Required:

* handle SIGINT / SIGTERM
* flush snapshot before exit
* stop hub cleanly

### Result

Clean restart safety

---

## 8. Atomic snapshot writes (IMPORTANT)

### Goal

Prevent corrupted JSON files

### Required pattern:

1. write `snapshot.tmp`
2. fsync
3. rename → `snapshot.json`

### Result

Crash-proof persistence layer

---

# PHASE 3 — Memory & Concurrency Stability

## 9. Full hub concurrency audit (FINAL PASS)

### Goal

Ensure zero hidden race conditions

### Validate:

* `clients` map usage
* `connToID`
* `idToClients`
* `lastAction`
* `Store`

### Rule:

✔ no map iteration without safe snapshot
✔ no mixed lock usage patterns

### Result

Production-safe concurrency model

---

## 10. Cleanup stale cooldown / lastAction entries

### Current state

✔ Cleanup function exists
❌ not reliably scheduled

### Required:

* periodic cleanup loop (e.g. every 10–60 min)
* remove inactive IDs

### Result

No memory growth over time

---

# PHASE 4 — Reliability Layer

## 11. WebSocket heartbeat (ping/pong)

### Goal

Detect dead connections early

### Required:

* periodic ping
* pong timeout handling

### Result

Reliable disconnect detection

---

## 12. Offline detection integration (frontend)

### Goal

Avoid useless reconnect attempts

### Required:

* `navigator.onLine`
* offline → immediate UI state switch

### Result

Better UX + fewer reconnection loops

---

## 13. Reconnect jitter (final tuning pass)

### Goal

Prevent synchronized reconnect storms

### Status

⚠️ partially implemented

### Required:

* random delay offset
* capped exponential backoff

### Result

Stable behavior at scale

---

# PHASE 5 — Performance & Scaling Foundations

## 14. Optimize snapshot generation

### Current state

✔ working
❌ full copy on every snapshot

### Improvements:

* reduce allocations
* optional delta snapshots
* immutable snapshot struct (future)

### Result

Lower CPU under heavy load

---

## 15. Binary WebSocket protocol (FUTURE OPTIMIZATION)

### Goal

Replace JSON with compact binary encoding

### Result

* lower bandwidth
* faster sync
* fewer allocations

---

## 16. Canvas chunking (MAJOR SCALING STEP)

### Goal

Support large worlds

### Concept:

* split canvas into regions
* sync only visible chunks

### Result

Massive scalability improvement

---

## 17. Per-client write isolation

### Goal

Prevent slow clients from affecting broadcast loop

### Required:

* per-client queues already exist (partially)
* enforce backpressure policy

### Result

Stable broadcast performance under load

---

# PHASE 6 — Feature Layer (OPTIONAL)

## 18. Pixel ownership tracking

Track:

* user ID
* timestamp

---

## 19. Timelapse / replay system

Replay canvas history from events

---

## 20. Heatmap analytics

Visualize activity intensity

---

## 21. Mobile support

* touch painting
* pinch zoom
* drag navigation

---

## 22. Social layer (optional)

* teams / factions
* collaborative painting groups

---

# 🚦 Immediate Execution Order (UPDATED)

## MUST DO FIRST (Critical stability)

1. Connection state machine (frontend)
2. Move “connected” to post-sync only
3. Reconnect backoff + jitter finalization
4. Atomic snapshot writes
5. Hub concurrency audit pass

---

## NEXT

6. Autosave loop
7. cleanup loop for lastAction
8. heartbeat ping/pong

---

## LATER

9. chunking
10. binary protocol
11. scaling architecture
