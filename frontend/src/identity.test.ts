import { test } from "node:test"
import assert from "node:assert/strict"
import { loadIdentity } from "./identity.ts"

test("blocked or unavailable storage keeps the canvas usable", () => {
  assert.equal(loadIdentity(null, () => "fresh-id").persistent, false)
  const storage = {
    getItem() {
      throw new Error("blocked")
    },
    setItem() {
      throw new Error("blocked")
    }
  }
  assert.deepEqual(
    loadIdentity(storage, () => "fresh-id"),
    { id: "fresh-id", persistent: false, repaired: false }
  )
})
test("invalid identities are replaced once and valid identities are reused", () => {
  let value = "invalid_id!"
  const storage = {
    getItem() {
      return value
    },
    setItem(_key: string, next: string) {
      value = next
    }
  }
  assert.equal(loadIdentity(storage, () => "fresh-id").repaired, true)
  assert.equal(loadIdentity(storage, () => "unused").id, "fresh-id")
})
