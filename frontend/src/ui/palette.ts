import { paletteEl } from "../dom"
import { config } from "../protocol"
import { state } from "../state"

export function setupPalette(): void {
  if (!paletteEl) return

  paletteEl.replaceChildren(...config.colors.map(color => { const button=document.createElement("button"); button.type="button"; button.className="color";button.dataset.color=color.value;button.style.backgroundColor=color.value;button.setAttribute("aria-label",color.name);return button }))

  const buttons = paletteEl.querySelectorAll<HTMLButtonElement>(".color")

  buttons.forEach((button) => {
    button.addEventListener("click", () => {
      const color = button.dataset.color
      if (color) {
        state.selectedColor = color
      }

      buttons.forEach((b) => b.classList.remove("active"))

      button.classList.add("active")
    })
  })

  buttons[0]?.classList.add("active")
}
