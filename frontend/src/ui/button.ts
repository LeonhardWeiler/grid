import { cooldownEl } from "../dom"
import { state } from "../state"
import { toCenteredCoords } from "../utils/gridCoords"
import { getCooldownRemaining } from "./cooldown"

export function updateButtonUI(): void {
  const feedback=document.getElementById("feedback")
  if(feedback && feedback.textContent!==state.message) feedback.textContent=state.message
  if (!cooldownEl) return
  if (state.pending) {cooldownEl.textContent="Saving pixel…";cooldownEl.classList.add("disabled");return}

  const remaining = getCooldownRemaining()

  if (remaining > 0) {
    cooldownEl.textContent = `Cooldown: ${(remaining / 1000).toFixed(1)}s`
    cooldownEl.classList.add("disabled")
    return
  }

  cooldownEl.classList.remove("disabled")

  if (!state.selectedPixel) {
    cooldownEl.textContent = "Select Pixel"
    return
  }

  const centered = toCenteredCoords(
    state.selectedPixel.x,
    state.selectedPixel.y
  )

  cooldownEl.textContent = `Click to accept Pixel (${centered.x} | ${centered.y})`
}
