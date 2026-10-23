# Grid

A shared 512 × 512 pixel canvas, live at [grid.coprime.tech](https://grid.coprime.tech).
Go, coder/websocket and framework-free TypeScript keep the application small.
The [project report](AGENT/project-health-report.html) records the audit and its implementation.

## Development and checks

```sh
nix develop
npm ci --ignore-scripts
go run .        # backend on 127.0.0.1:4000
npm run dev     # Vite on localhost:5173; proxies /ws
```

Use two terminals. The supported toolchain is pinned in `flake.lock`: Go 1.27.3,
Node 24 and Chromium. Dependencies use `go.sum` and `package-lock.json`.

```sh
nix build -L                    # frontend tests/typecheck; Go vet and race tests
nix develop -c npm run test:browser  # isolated production server + Chromium/axe
nix develop -c go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
nix develop -c npm audit
nix develop -c gitleaks git --redact --config .gitleaks.toml
```

Browser tests use temporary state and never write to production. They cover real
broadcasts and confirmations, uncertain writes, reconnect/offline recovery, version
gaps, blocked storage, keyboard use, touch pan/pinch, DPR, layout and accessibility.
`BROWSER_EXECUTABLE` can select another installed Chromium. `nix develop` sets it.
The security workflow also runs daily; Dependabot proposes dependency/action updates.

After npm dependency changes, recompute `npmDepsHash` in `flake.nix` with
`nix run nixpkgs#prefetch-npm-deps -- package-lock.json`, or temporarily use
`pkgs.lib.fakeHash` and use the hash reported by a Nix build. Frontend formatting:
`nix run nixpkgs#prettier -- --write frontend tests '*.config.ts' tsconfig.json`.

## Drawing and protocol

Choose a named color, select a cell, then Place. Drag to pan, scroll or pinch to
zoom. The original floating layout stays minimal. Open the top-right coordinate readout
(or press `?`) for exact coordinates, zoom/reset controls, help and privacy information.
With the canvas focused, arrow keys select cells, Shift moves ten cells, and
Enter/Space places. Confirmation is explicit; rejected or uncertain writes retain
the selection and are never retried automatically.

[`internal/protocol/config.json`](internal/protocol/config.json) defines the
protocol version, canvas size and palette for both languages. The server owns
state. A new connection always receives a validated full snapshot; live events
and recovery deltas have contiguous increasing versions. See
[the protocol contract](AGENT/protocol.md). Rendering uses one incremental raster,
shared with the minimap, and stops scheduling frames while idle.

Every accepted write is durably journaled before broadcast. Browser IDs have a
five-second wait; changing IDs cannot bypass the persisted shared-network budget
(burst three, sustained one pixel per five seconds). IPv6 uses a /64 network.
Global drawing is bounded at ten writes/second with a burst of twenty. These
limits reduce abuse without accounts; distributed abuse still requires operator
capacity controls. There are at most 256 connections, twelve per network, bounded
handshake/message/sync rates, and 32 queued messages/8 MiB per connection. Slow
clients disconnect and recover through a new snapshot.

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `STATE_DIRECTORY` | `./data` | Snapshot, journal and exclusive process lock. |
| `STATIC_DIR` | `./public/dist` | Vite production output. |
| `LISTEN_ADDR` | `127.0.0.1:4000` | Public HTTP/WebSocket listener behind nginx. |
| `ADMIN_ADDR` | disabled | Optional literal loopback listener for `/metrics` and `/backup`. |
| `TRUST_PROXY` | unset | Set to `1` only behind a loopback proxy that overwrites `X-Real-IP`. |
| `GOMEMLIMIT` | Go default | Production uses `192MiB` below systemd's 256 MiB limit. |

Public routes: `/healthz`, `/readyz`, `/version`, `/ws` and static assets.
Storage failures make readiness return 503 and trigger a restart while retaining
the journal. Private operations routes are absent from the public listener.
CSP and other browser protections apply to every HTTP response; HTML is
revalidated and content-hashed assets are cached immutably. Production adds HSTS.

## Deployment and operation

A push to `main` builds/tests the locked package and runs browser checks, then
triggers the **waentw** server configuration pipeline (GitLab project 87242023)
with the exact `GITHUB_SHA` as `GRID_REVISION`. That pipeline serializes server
switches. Both pipelines wait for the expected public `/version` and a successful
read-only HTTP/WebSocket snapshot probe. A failed or merely queued deployment
cannot report success.

From a clean, committed waentw checkout:

```sh
GRID_REVISION=<full-tested-commit> nix run .#deploy
nix run .#deploy                 # deploy existing locked inputs
nix run .#deploy -- --update     # intentional update of all application/OS inputs
```

An exact Grid deployment updates only Grid. It preserves the locked OS and other
applications. The server configuration owns nginx, the systemd sandbox, backups
and log retention. The NAS configuration owns the restricted upload destination
and independent two-minute HTTP/WebSocket monitor, with existing NAS failure
notifications. Do not place deploy tokens or SSH private keys in these repos.

```sh
curl --fail https://grid.coprime.tech/readyz
./result/bin/grid -probe https://grid.coprime.tech  # read-only HTTP + WS init
ssh deploy@server systemctl status grid grid-backup.timer
ssh deploy@server journalctl -u grid -f
ssh deploy@server curl -fsS http://127.0.0.1:4001/metrics
```

NixOS sends SIGHUP on upgrades. Drawing continues until thirty seconds without a
new write, waiting at least thirty seconds and at most five minutes. The server
then saves, finishes bounded WebSocket close handshakes and exits. systemd starts
the new package; clients reconnect without reloading the page. SIGTERM/SIGINT
stop immediately after saving. Roll back with
`ssh deploy@server sudo nixos-rebuild switch --rollback`; state remains in place.
Snapshot schema 1 remains compatible with legacy schema 0; an application rollback
must support the current state/protocol. Test older releases in isolated state.

## Persistence, backup and restore

Exactly one process owns a state directory. Accepted pixels and temporary rate
limit metadata are appended and fsynced to `snapshot.json.wal`. Every fifteen
seconds a checkpoint copies state under its lock, encodes outside that lock,
atomically replaces an fsynced snapshot and retains newer WAL entries. Unchanged
checkpoints do no disk work. Startup validates the snapshot and replays contiguous
newer entries. A truncated last journal line is discarded; complete corruption
fails startup. A failed journal is never compacted away. Five-second storage
watchdogs fail readiness; the service restarts within a bounded shutdown window.
Persistent disk errors still require repair; they are not hidden by restarts.

The hourly backup job downloads a consistent private `/backup` export, validates
it with `grid -check-state`, and uploads only pixels and the version to
`/data/appdata/grid` on the NAS. The dedicated server key is loaded by systemd
from `/var/lib/grid-backup-key/id_ed25519`; the NAS host key is pinned. That key
cannot open a shell, forward ports, read, delete or overwrite previous backups.
The NAS expires active upload files after approximately thirty days. Its existing
filesystem snapshots can retain copies for twelve months; the external backup
disk also keeps yearly archives without an age limit. These archives contain
only the public drawing, never browser/network IDs. Backup failures and a backup older than two
hours fail monitored systemd units. This provides a one-hour backup interval;
total source/backup loss between runs can lose changes since the last upload.

Restore into an **empty temporary directory** first, using an administrator's
NAS access, not the write-only upload key:

```sh
mkdir -m 700 /tmp/grid-restore
scp root@nas:/data/appdata/grid/snapshot-<timestamp>.json /tmp/grid-restore/snapshot.json
./result/bin/grid -check-state /tmp/grid-restore
STATE_DIRECTORY=/tmp/grid-restore STATIC_DIR=./result/share/grid LISTEN_ADDR=127.0.0.1:4010 ./result/bin/grid
```

Inspect the local canvas/version, then stop that test process. For an intentional
production restore, stop `grid.service`, preserve its entire current state
separately, install the validated snapshot as mode 0600 into `/var/lib/grid`,
remove only the production WAL after preserving it, and start Grid. Never merge
an unrelated snapshot with the existing WAL. Do not run recovery tools on a live
state directory. Copying an actively changing snapshot/WAL pair is not a backup;
use the private export or stop the service.

The Privacy dialog documents the local random ID, temporary pseudonymous rate
limits, public drawing, ID reset, backup contents and technical log retention.
