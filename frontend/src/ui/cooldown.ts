import { requestFrame } from "../frame.ts"
let cooldownEnd = 0

// A duration from the server avoids comparing clocks on different devices.
// The monotonic clock also tolerates adjustments to the device's wall clock.
export function setCooldown(remainingMs: number): void {
  cooldownEnd = performance.now() + Math.max(0, remainingMs)
  requestFrame()
}

export function isCooldownActive(): boolean {
  return getCooldownRemaining() > 0
}

export function getCooldownRemaining(): number {
  return Math.max(0, cooldownEnd - performance.now())
}
