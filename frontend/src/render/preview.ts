import { ctx } from "../dom"
import { state } from "../state"
import type { Pixel } from "../model"
import { worldToScreen } from "../utils/coords"

export function drawPreview(): void {
  if (state.hoverPixel) {
    const pos: { x: number; y: number; size: number } = worldToScreen(
      state.hoverPixel.x,
      state.hoverPixel.y
    )

    ctx.fillStyle = "rgba(255,255,255,0.2)"
    ctx.fillRect(pos.x, pos.y, pos.size, pos.size)
  }

  if (state.selectedPixel) {
    const pos: { x: number; y: number; size: number } = worldToScreen(
      state.selectedPixel.x,
      state.selectedPixel.y
    )

    ctx.save()
    ctx.globalAlpha = 0.35
    ctx.fillStyle = state.selectedColor
    ctx.fillRect(pos.x, pos.y, pos.size, pos.size)
    ctx.restore()
    ctx.strokeStyle = "black"
    ctx.lineWidth = 4
    ctx.strokeRect(pos.x, pos.y, pos.size, pos.size)
    ctx.strokeStyle = "white"
    ctx.lineWidth = 2
    ctx.strokeRect(pos.x, pos.y, pos.size, pos.size)
  }
}
