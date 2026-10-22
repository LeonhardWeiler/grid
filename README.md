# grid

A minimal real-time pixel canvas inspired by r/place.

Built with:

* Go
* WebSockets
* TypeScript
* HTML5 Canvas API
* Vite (frontend tooling)
* Nix Flakes (optional)
* In-memory state with event log + snapshot persistence

---

## Features

* Real-time pixel synchronization
* Server-authoritative canvas state
* Event-based sync (delta updates)
* Snapshot fallback sync
* Lightweight WebSocket protocol (JSON)
* Reconnect with backoff + jitter
* Rate limiting (cooldown per user)
* Minimal frontend (no framework)
* Stateless clients
* Crash-safe persistence (snapshot-based)

---

## Architecture

### Backend

* Go
* `github.com/coder/websocket`
* `net/http`
* In-memory pixel store (event log + ring buffer)
* Snapshot persistence (JSON file)
* Hub-based WebSocket broadcaster

### Frontend

* TypeScript
* Vite
* HTML5 Canvas API
* Native WebSocket API
* Connection state machine (sync-aware UI)

---

## Getting Started

### With Nix (recommended)

```bash
git clone https://github.com/LeonhardWeiler/grid.git
cd grid

nix develop

go mod tidy
air
npm i
bun run dev
```

Open:

```text
http://localhost:5173 (frontend)
http://localhost:4000 (backend)
```

---

### Without Nix

Requirements:

* Go 1.24+
* Bun

Install dependencies:

```bash
go mod tidy
cd frontend
bun install
```

---

## Development

Run backend:

```bash
air
```

Run frontend:

```bash
bun run dev
```

---

## Build

### Frontend build

```bash
bun run build
```

Output:

```text
public/dist
```

---

### Full package build (Nix)

```bash
bun run package
```

---

## Current Status

### Core System

* [x] WebSocket server (Go)
* [x] Real-time pixel sync
* [x] Server-authoritative state
* [x] Event-based delta sync
* [x] Snapshot fallback sync
* [x] Rate limiting / cooldown system
* [x] Client reconnection logic (backoff + jitter)
* [x] Connection state machine (frontend)
* [x] Canvas rendering system (HTML5 Canvas)

---

### Persistence

* [x] In-memory pixel store
* [x] Snapshot persistence (JSON)
* [x] Startup state recovery
* [ ] Atomic snapshot writes (tmp → rename)
* [ ] Graceful shutdown persistence
* [ ] Periodic autosave loop

---

### Reliability

* [x] Reconnect handling
* [x] Offline detection (basic)
* [ ] WebSocket heartbeat (ping/pong)
* [ ] Full concurrency audit (final pass)
* [ ] Cleanup of stale lastAction entries

---

### Performance

* [x] Efficient event buffer (ring buffer)
* [ ] Snapshot optimization (reduce full copies)
* [ ] Message batching (optional)
* [ ] Binary WebSocket protocol (future)

---

### Scaling (Future)

* [ ] Canvas chunking system
* [ ] Multi-server architecture
* [ ] Redis PubSub (optional)
* [ ] Horizontal scaling support

---

### Deployment

* [ ] Auto deployment via SSH
* [ ] Production Docker setup (optional)

---

## Design Goals

* Simplicity over abstraction
* Minimal dependencies
* Real-time consistency
* Server authority (clients are dumb)
* Low-latency pixel sync
* Easy local development
* Hackable architecture

---

## Future Ideas

* Pixel ownership / history tracking
* Replay / timelapse system
* Heatmap analytics
* Mobile touch controls
* Collaborative teams / factions
* Custom maps (e.g. regional / thematic boards)
* Competitive events / time-limited canvases
