import assert from "node:assert/strict"
import { test } from "node:test"
import { clampView } from "./bounds.ts"

for (const [width, height] of [
  [390, 844],
  [1920, 1080],
  [5120, 1440],
  [390, 3000],
  [50000, 5000]
] as const) {
  test(`viewport stays inside all grid edges at ${width}×${height}`, () => {
    for (const zoom of [0.001, 0.08, 0.6, 10]) {
      for (const [x, y] of [
        [-10000, -10000],
        [10000, 10000],
        [256, 256]
      ] as const) {
        const view = clampView({ x, y, zoom }, width, height, 512, 40)
        const halfWidth = width / (2 * 40 * view.zoom)
        const halfHeight = height / (2 * 40 * view.zoom)
        assert(view.x - halfWidth >= -1e-9)
        assert(view.x + halfWidth <= 512 + 1e-9)
        assert(view.y - halfHeight >= -1e-9)
        assert(view.y + halfHeight <= 512 + 1e-9)
      }
    }
  })
}

test("ordinary camera position and zoom are preserved", () => {
  const view = { x: 100, y: 250, zoom: 0.6 }
  assert.deepEqual(clampView(view, 1280, 800, 512, 40), view)
})

test("resizing a zoomed-out view raises the minimum zoom", () => {
  const small = clampView({ x: 500, y: 500, zoom: 0.01 }, 390, 844, 512, 40)
  const wide = clampView(small, 5120, 1440, 512, 40)
  assert(wide.zoom > small.zoom)
  assert.equal(wide.x, 256)
})
