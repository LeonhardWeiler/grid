# Restore the original minimal UI, 2026-10-23

User requested the same minimal appearance as before the audit implementation.
Reference: commit `0e38543` and `AGENT/audit/2026-10-22/desktop.png`.

Restored the full-window canvas, original vertical palette, 250px desktop minimap,
corner coordinate readout, centered coordinates, bottom confirmation field,
0.6 initial zoom and yellow selection outline. No persistent header/footer panels
or onboarding text. Extra keyboard/coordinate/zoom/privacy controls are in a
modal opened from the original coordinate field or the `?` shortcut.

Retained native labelled buttons, keyboard and touch input, focus handling,
explicit ACK/rejection behavior, storage fallback, rate limits, all backend
security/persistence work, DPR rendering and the incremental raster. Transparent
44px hit areas surround the original 34px color swatches. The minimap shrinks on
small screens; the palette moves horizontally only on short screens so it stays
reachable. The minimap uses area filtering so isolated pixels survive reduction.

The 17 browser tests passed, including axe on four viewports, DPR 1/2/3, 200% text,
real WS acceptance/broadcast, reconnect/gaps and a new geometry check against the
original desktop dimensions. Desktop/mobile/landscape captures use a local fake
socket seeded with the already public pre-release drawing; production was not
painted during verification.
