# CLAUDE.md

grid: a real-time pixel canvas like r/place. Go WebSocket backend,
TypeScript + Vite frontend on HTML5 Canvas. Details in `README.md`.

## Commands

From the repo root:

```bash
air              # backend with live reload, :4000
bun install
bun run dev      # Vite dev server, :5173
bun run build    # production build
```

## Layout

- `main.go`, `internal/` - server: `hub.go` (broadcast), `store.go` (pixel grid + event ring buffer),
  `persistence.go` (JSON snapshot), `autosave.go`, `validate.go`, `ws.go`, `config.go`.
- `frontend/` - Canvas renderer and WebSocket client.

## Rules

- The server is the single source of truth. Clients send intents, never state.
- Every incoming message is validated and rate-limited per user.
- Messages are minimal JSON. Deltas for live updates, a snapshot for reconnect.
- No frontend framework: Canvas + TypeScript only.
- A change to rate limiting, the sync format or the snapshot model changes backend and frontend together.
