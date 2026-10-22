import { cooldownEl } from "../dom"
import { state, connection } from "../state"
import { getCooldownRemaining } from "./cooldown"
import { config } from "../protocol"
import { pixels } from "../model"

function setText(element: HTMLElement | null, text: string) {
  if (element && element.textContent !== text) element.textContent = text
}
export function updateButtonUI(): void {
  setText(document.getElementById("feedback"), state.message)
  const remaining = getCooldownRemaining()
  const pixel = state.selectedPixel
  const colorName =
    config.colors.find((c) => c.value === state.selectedColor)?.name ?? "color"
  let label = "Select a pixel"
  if (state.pending) label = "Saving pixel…"
  else if (connection.status !== "connected") label = "Waiting for connection"
  else if (remaining > 0)
    label = "Ready in " + (remaining / 1000).toFixed(1) + "s"
  else if (pixel)
    label = "Place " + colorName + " at " + pixel.x + ", " + pixel.y
  setText(cooldownEl, label)
  const disabled =
    connection.status !== "connected" ||
    state.pending !== null ||
    !pixel ||
    remaining > 0
  if (cooldownEl.disabled !== disabled) cooldownEl.disabled = disabled
  if (pixel) {
    const current = pixels.get(pixel.x + ":" + pixel.y)
    const currentName =
      config.colors.find((c) => c.value === current)?.name ?? "Empty"
    setText(
      document.getElementById("selection"),
      "Pixel " +
        pixel.x +
        ", " +
        pixel.y +
        ": " +
        currentName +
        ". Selected color: " +
        colorName +
        "."
    )
  } else
    setText(
      document.getElementById("selection"),
      "Select a pixel to inspect its color."
    )
}
