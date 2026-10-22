# Protocol and state contract

`internal/protocol/config.json` is embedded by Go and imported by TypeScript.
No code generator or runtime configuration service is needed. Protocol changes
must update both endpoints and their tests together. Public coordinates are
integers from zero through `gridSize - 1`; colors are exact configured values.

Client messages:

- `init_client`: required `clientId` (1–64 ASCII letters/digits/hyphens), `lastVersion`.
- `sync`: required nonnegative integer `lastVersion`.
- `set_pixel`: required integer `x`, `y`, configured `color`; optional `requestId`
  with the same syntax as IDs. Current clients always send a random request ID.

A new connection receives `init` with protocol, gridSize, pixels keyed as `x:y`,
version and `cooldownMs`. A reconnect always receives a full authoritative canvas.
Each accepted mutation increments version once; `pixel_update` broadcasts to all
initialized clients. `accepted`/`rejected` carries the originating `requestId`,
remaining cooldown and accepted version or rejection code. Legacy clients without
a request ID still get pixel/cooldown events. An ACK confirms durable acceptance,
not that other clients have read it. Unknown outcomes must not auto-retry.

`sync` replies contain contiguous events after the requested version, the final
version and cooldown. An empty current delta contains no events. Requests older
than the bounded 100,000-event ring receive `init`. Gaps disable placing while a
single recovery request runs; invalid/noncontiguous responses reconnect from an
authoritative init. Routine polling is twenty seconds, recovery requests are
spaced by at least 1.05 seconds, and unanswered operations expire after ten seconds.
Server `error` messages carry a retry duration for a temporarily busy sync budget.

Snapshot schema 1 contains pixels, version and temporary browser/source budgets.
Legacy schema 0 is read-compatible. Independent backups omit both identity maps.
The WAL includes durable per-event metadata to preserve limits across restart.
One process owns the state-directory kernel lock. Hub mutation ordering is
`stateMu`, then hub/store locks; `saveMu` serializes checkpoints, and disk encoding
runs after releasing `stateMu`. Keep state mutation behind Hub operations.
