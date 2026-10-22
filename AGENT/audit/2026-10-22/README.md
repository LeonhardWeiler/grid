# Audit evidence, 2026-10-22

This directory supports [the project health report](../../project-health-report.html).
The report is written in German to match the audit request and the previous report.

Baseline: `0e38543cfacf934385717e551a03dda497aeea68`, initially clean checkout.
The application and production configuration were not changed. Production checks
used HTTP requests, WebSocket initialization and small invalid-message probes;
they did not send `set_pixel`, restart services or change configuration.

## Recorded evidence

- `browser-results.json`: Chromium 155, Playwright and axe results; layout,
  storage failure and synthetic full-canvas timings. Seven synchronous redraws
  per population, 1440 by 900 CSS pixels, Linux/Ryzen 7 6800H. No CPU/network
  throttling. These are local laboratory observations, not field performance
  or real-device Safari/Android results.
- `extra-results.json`: landscape button occlusion, a one-second idle DOM
  mutation count, and redacted context for two historical development/test
  `secret_key_base` scanner findings. No secret values are included.
- `desktop.png`, `mobile.png`, `mobile-landscape.png`: production UI screenshots
  at 1440x900, 390x844 and 844x390. No pixel was confirmed during capture.
- `protocol-results.json`: read-only production snapshot counts and invalid
  JSON/oversize-message handling; the write failure and health result come from
  a **separate local instance** on port 4400 with temporary state.
- `backend-results.txt`: isolated Go probes and a second benchmark run, five
  iterations each. An initial run measured about 1.68 ms for an empty full-ring
  delta, 457 ms for snapshot+JSON and 324 ms for Save. The archived second run
  measured about 2.01 ms, 446 ms and 453 ms. Allocation counts vary with JSON
  buffer reuse and GC. Neither run measures production throughput or p99.
- `reproduce_test.go.txt`: laboratory tests/benchmarks, intentionally not added
  to the application's executable test suite. They demonstrate current behavior;
  passing these probes means the observation reproduced, not that the defect is fixed.
- `npm-audit.json`: npm audit response for the locked dependency tree, including
  development/build dependencies. Severity reflects scanner metadata; deployment
  applicability is assessed separately in the report.
- `govulncheck-retry.txt`: successful source scan with Go 1.26.2. The first run
  failed to resolve vuln.go.dev; one bounded retry succeeded. Static symbol
  reachability does not prove that every TLS/HTTP2/Windows prerequisite is enabled.
- `report-validation.json`: verification of the report itself: 31 cards, matching
  severity counts, working filters, valid local links/images and fragment targets,
  no axe violations, no page errors and no document overflow at 390 CSS pixels.

Live server details were read using `ssh -o BatchMode=yes deploy@server` and
noninteractive read-only `sudo` commands: `systemctl show/cat`, `journalctl`,
`systemd-analyze security`, `ss`, `ps`, disk/memory statistics, active nginx and
logrotate configurations, `iptables-save`, and `ip6tables-save`. Provider backup
settings, tailnet ACLs, external monitoring, and Git hosting protection settings
were not available. Raw host dumps, IP-level access logs, notification URLs,
production snapshots and credentials are deliberately not stored here.

The T3 preview browser failed with missing host libraries. The available Nix
Chromium binary was used with temporary Playwright and axe installations under
`/tmp/grid-audit`, leaving package.json and package-lock.json unchanged.

## Repeat the standard gates

From the repository root:

```sh
nix develop -c sh -c 'go vet ./... && go test -race -coverprofile=/tmp/grid-cover.out ./... && npm test && npx tsc --noEmit && npm run build'
nix build -L
nix develop -c npm audit --json
nix develop -c go run golang.org/x/vuln/cmd/govulncheck@latest ./...
nix shell nixpkgs#gitleaks -c gitleaks git --redact --no-banner .
```

The security tools return nonzero when findings are present. Online advisory
databases change over time; repeat the applicability review when fixing findings.
Gitleaks scanned 111 commits (113 reachable in Git) and found two entries in
historical Phoenix development/test configuration at commit `ab5ac9d`; no current
Go production use of those values was identified.

## Repeat isolated backend probes

Run from the project root. This copies the baseline into a newly created temporary
directory, so writes remain separate from both the repository and production.

```sh
audit_source_root="$PWD"
audit_scratch_dir="$(mktemp -d /tmp/grid-audit-repro.XXXXXX)"
cp go.mod go.sum "$audit_scratch_dir/"
cp -r internal "$audit_scratch_dir/"
cp AGENT/audit/2026-10-22/reproduce_test.go.txt "$audit_scratch_dir/internal/audit_test.go"
cd "$audit_scratch_dir"
nix develop "$audit_source_root" -c go test -v -run TestAudit -bench BenchmarkAudit -benchtime=5x -count=1 ./internal
```

The copied original `internal/reliability_test.go` provides `openTestHub` used by
the audit probes. The probes test identity changes, omitted coordinates,
semantically invalid snapshots, write-failure recovery, full-ring empty deltas,
full-canvas snapshot encoding and full-canvas Save. No external server is contacted.

For frontend measurement, start a separate local Vite instance, intercept its
WebSocket in an isolated browser context with an empty authoritative init,
import `/src/render/render.ts`, populate its exported pixel Map with 10, 10,000,
100,000 and 262,144 synthetic entries, then time `requestRender(); render()`
seven times per population using `performance.now()`. Do not insert synthetic
pixels into production. Match the viewport, browser, hardware and colors when
comparing the recorded values. Report median and range, not just the fastest run.

Primary advisory and nginx documentation links are included next to the relevant
claims in the report. The original report is preserved in Git history; its old
IDs are reconciled individually in the new report.
