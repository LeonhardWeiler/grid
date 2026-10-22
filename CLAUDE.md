# Grid contributor guide

Read `README.md` for current development, deployment, recovery and checks, and
`AGENT/protocol.md` for the shared protocol and locking contract.

- Use `nix develop`, `npm ci --ignore-scripts`, `go run .` and `npm run dev`.
- Run `nix build -L` and `nix develop -c npm run test:browser` for release changes.
- The server owns all canvas state. Validate intents and keep bounded queues/budgets.
- Keep the framework-free TypeScript UI and JSON/file persistence simple.
- Edit shared grid/palette constants in `internal/protocol/config.json`.
- Change both endpoints together when changing protocol or snapshot invariants.
- Production state and credentials are outside Git and immutable packages.
- The project report tracks audit findings; do not revive obsolete TODO lists.
