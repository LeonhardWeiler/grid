import { canvas, ctx ,view} from "../dom"
import { drawMinimap } from "./minimap"
import { drawGrid } from "./grid"
import { drawPixels } from "./pixels"
import { drawPreview } from "./preview"

export { pixels } from "../model"

let needsRender = true

export function requestRender(): void {
  needsRender = true
}

export function render(): void {
  if (!needsRender) return

  ctx.fillStyle = "#111"
  ctx.fillRect(0, 0, view.width, view.height)

  drawPixels()
  drawGrid()
  drawPreview()
  drawMinimap()

  needsRender = false
}
