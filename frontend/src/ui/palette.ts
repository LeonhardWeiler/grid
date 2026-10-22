import { paletteEl } from "../dom"
import { config } from "../protocol"
import { state } from "../state"
import { requestRender } from "../render/render"

export function setupPalette(): void {
  const buttons = config.colors.map((color) => {
    const button = document.createElement("button")
    button.type = "button"
    button.className = "color"
    button.dataset.color = color.value
    button.style.backgroundColor = color.value
    button.setAttribute("aria-label", color.name)
    button.setAttribute(
      "aria-pressed",
      String(state.selectedColor === color.value)
    )
    button.classList.toggle("active", state.selectedColor === color.value)
    button.addEventListener("click", () => {
      state.selectedColor = color.value
      for (const other of buttons) {
        const active = other === button
        other.classList.toggle("active", active)
        other.setAttribute("aria-pressed", String(active))
      }
      requestRender()
    })
    return button
  })
  paletteEl.replaceChildren(...buttons)
}
