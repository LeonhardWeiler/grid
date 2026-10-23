# Implementation verification, 2026-10-22

The report's original audit is retained under `AGENT/audit/2026-10-22` and in Git.
This folder records the changed implementation. Each finding ID is included in
its commit subject, including the adjacent waentw server and NAS repositories.
The report links the original defect to the implementation and its checks.

Reproduce in an isolated checkout:

```sh
nix build -L
nix develop -c npm run test:browser
nix develop -c go test ./internal -run TestDenseState -v
nix develop -c go test ./internal -run NONE -bench 'CurrentDelta|DenseSnapshot' -benchmem
nix develop -c go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
nix develop -c npm audit --json
nix develop -c gitleaks git --redact --config .gitleaks.toml
```

Browser checks run a real production HTTP/WebSocket server in temporary state;
network failure and protocol edge cases use deterministic intercepted sockets.
Four viewport sizes run axe, DPR 1/2/3 checks exercise coordinate selection,
and CDP touch input checks pan/pinch without unintended selection. Physical
mobile devices and an interactive screenreader were not available.

`raster-benchmark.json` records direct render-call CPU time on a synthetic local
canvas (Ryzen 7 6800H, Chromium 155); it is not end-to-end GPU/frame timing.
No synthetic canvas or load flood was sent to production. Live probes are
read-only; a backup restore is validated in empty temporary state, not over the
production drawing. Exact deployed revisions, restore time/version and service
verification are recorded after release.

Live verification completed on 2026-10-23. See `live-verification.json`,
`restore.json`, live response headers and desktop/mobile captures. The functional
release retained all eight pixels at version thirteen. The restricted upload key
was tested against an isolated disposable marker, not against backup history.
Restore took 1.882 seconds including NAS download and a fresh local HTTP/WS server.

The first server switch exposed an nginx log-format ordering mistake and briefly
stopped the proxy. It was repaired immediately; target-side `nginx -t` now runs
before activation. The pure-evaluation deploy error and alert suppression were
also corrected. The latest server pipeline and initial GitHub release are green.
Documentation, minimum-Go-version and CI-maintenance follow-ups use the same
SHA/readiness verification; the public `/version` shows the exact active revision.

Retention has two layers: the Grid upload folder expires files after about thirty
days; existing NAS filesystem snapshots retain appdata for up to twelve months,
and external-disk yearly archives have no age limit. Those longer-lived copies
contain only public pixels/version. The Privacy dialog and operating guide make
this distinction explicit; no existing NAS snapshots were modified or deleted.
