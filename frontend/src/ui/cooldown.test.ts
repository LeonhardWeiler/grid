import assert from "node:assert/strict"
import { test } from "node:test"
import { setCooldown, getCooldownRemaining, isCooldownActive } from "./cooldown.ts"

test("countdown expires after the server duration despite device clock differences", t => {
  let elapsed = 100
  t.mock.method(performance, "now", () => elapsed)
  t.mock.method(Date, "now", () => 9000000000000)
  setCooldown(5000)
  assert.equal(getCooldownRemaining(), 5000)
  assert(isCooldownActive())
  elapsed += 2000
  assert.equal(getCooldownRemaining(), 3000)
  elapsed += 3000
  assert.equal(getCooldownRemaining(), 0)
  assert(!isCooldownActive())
})

test("reconnect restores the remaining cooldown and expired cooldowns clear it", t => {
  t.mock.method(performance, "now", () => 500)
  setCooldown(1800)
  assert.equal(getCooldownRemaining(), 1800)
  setCooldown(-100)
  assert.equal(getCooldownRemaining(), 0)
})
