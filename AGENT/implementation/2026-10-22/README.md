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
