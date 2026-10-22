# Grid

A real-time 512 × 512 pixel canvas built with Go, coder/websocket, TypeScript
and Vite. Live at **https://grid.coprime.tech**.

The server validates coordinates and colors and enforces a five-second cooldown
per browser identity. The countdown uses the server-reported remaining duration
and a monotonic browser timer, so device clock differences do not affect it.
All tabs share the identity in `localStorage`. The server
owns the canvas; each connection starts with a complete snapshot, followed by
ordered live updates. Zoom and pan stay inside the canvas at every window size.
Heartbeats, gap recovery and automatic reconnect keep
clients synchronized after sleep, network loss and deployments.

## Development

```sh
nix develop
npm ci
go run .        # backend, 127.0.0.1:4000
npm run dev     # frontend, localhost:5173 (proxies /ws to the backend)
```

Use two terminals for the servers. Without Nix, install Go 1.26.2+ and Node 24+.
For a production build and all checks:

```sh
nix build -L
STATE_DIRECTORY=./data STATIC_DIR=./result/share/grid ./result/bin/grid
```

`nix build` builds the frontend from `package-lock.json`, tests and typechecks it, runs
Go vet and the Go tests with the race detector, and packages both the binary
(`bin/grid`) and frontend (`share/grid`). `flake.lock`, the npm dependency hash
and the Go vendor hash pin the complete build. After dependency changes, update
the corresponding hash in `flake.nix` using the hash reported by `nix build`.

## Production deployment

A push to `main` runs `.github/workflows/deploy.yml`: build and test the locked
Nix package, then trigger the existing **waentw server configuration** pipeline
(GitLab project 87242023), using the repository secret `GITLAB_TRIGGER_TOKEN`.
The server configuration's deploy command updates the `grid` input alongside
`slf`, commits the lock locally when run manually, and switches NixOS on
`deploy@server` through Tailscale. GitLab serializes deployments with the
`server` resource group.

The server configuration lives in the waentw repository:

- `flake.nix`: Grid input and deployment command.
- `server-config/configuration.nix`: `grid.service` and the HTTPS nginx virtual
  host for `grid.coprime.tech`, including WebSocket proxying and ACME.

The app listens only on `127.0.0.1:4000`. systemd manages its unprivileged user,
automatic startup/restart and persistent `/var/lib/grid` state directory.
State is outside the immutable package and survives upgrades and rollbacks.

To deploy manually from the committed waentw checkout:

```sh
nix run .#deploy
```

For an exact locked rebuild without updating inputs, use
`nixos-rebuild switch --flake .#server --target-host deploy@server --sudo`.

### Updating an active canvas

NixOS reloads the existing process with `SIGHUP` instead of immediately killing
connections. It continues serving and accepting drawings until there have been
30 seconds without a new pixel, waiting at least 30 seconds and at most five
minutes. Then it saves, closes connections, and exits; systemd starts the new
package. Open clients keep their camera, selection and identity, reconnect with
backoff starting at 250 ms, and resynchronize without reloading the page.
There is a brief reconnect during the process change; uninterrupted WebSocket
handover between processes is not implemented. `SIGTERM`/`SIGINT` save and exit
immediately for an explicit service stop or server shutdown.

### Persistence and recovery

Each accepted pixel, its version and the writer's cooldown are appended to
`snapshot.json.wal` and **fsynced before broadcasting**. Every 15 seconds and on
shutdown, an fsynced snapshot is atomically renamed into place, its directory
is synced, and the journal is compacted. Startup loads the snapshot and replays
newer journal entries. Old snapshots containing only `pixels` and `version`
remain compatible. A truncated final journal record after a crash is discarded;
a complete corrupt record or unreadable snapshot causes startup to fail instead
of silently starting an empty canvas.

Back up both `snapshot.json` and `snapshot.json.wal` together, ideally with the
service stopped or a filesystem snapshot. Keep independent backups: persistence
on the server does not protect against disk loss.

```sh
ssh deploy@server systemctl status grid
ssh deploy@server journalctl -u grid -f
curl --fail https://grid.coprime.tech/healthz
ssh deploy@server sudo systemctl reload grid   # exercise a graceful restart
```

To roll back the system package:
`ssh deploy@server sudo nixos-rebuild switch --rollback`. The state directory
stays in place.

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `STATE_DIRECTORY` | `./data` | Persistent snapshot and journal directory; systemd sets `/var/lib/grid`. |
| `STATIC_DIR` | `./public/dist` | Built frontend directory. |
| `LISTEN_ADDR` | `127.0.0.1:4000` | HTTP and WebSocket listen address. |

## Verification

```sh
nix develop -c sh -c 'go vet ./... && go test -race ./... && npm test && npx tsc --noEmit && npm run build'
```

Tests cover snapshot + journal recovery without shutdown, partial writes,
corruption detection, cooldown recovery, concurrent persistence, delta fallback,
full WebSocket initialization and cross-client broadcasts, cooldown countdowns
with skewed device clocks, and viewport bounds on mobile and wide screens.
