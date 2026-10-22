import { test } from "node:test"
import assert from "node:assert/strict"
import { applyEvents } from "./sync.ts"

test("a gap cannot partially change or advance the local canvas", () => {
  const pixels = new Map([["0:0", "#ffffff"]])
  assert.throws(() =>
    applyEvents(pixels, 1, [{ version: 3, x: 1, y: 1, color: "#ff0000" }], 3)
  )
  assert.deepEqual([...pixels], [["0:0", "#ffffff"]])
})
test("duplicate and empty deltas do not redraw or roll back the version", () => {
  const pixels = new Map<string, string>()
  assert.deepEqual(applyEvents(pixels, 3, [], 2), {
    version: 3,
    changed: false
  })
  assert.deepEqual(applyEvents(pixels, 3, [], 3), {
    version: 3,
    changed: false
  })
  assert.deepEqual(
    applyEvents(
      pixels,
      3,
      [
        { version: 3, x: 0, y: 0, color: "#ffffff" },
        { version: 4, x: 1, y: 1, color: "#ff0000" }
      ],
      4
    ),
    { version: 4, changed: true }
  )
})
