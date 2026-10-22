import { ctx, canvas } from "../dom"
import { pixels } from "../model"
import { worldToScreen } from "../utils/coords"

export function drawPixels(): void {
  for (const [key, color] of pixels) {
    const parts = key.split(":").map(Number)
    if (parts.length !== 2) continue
    const xVal = parts[0]
    const yVal = parts[1]
    if (xVal === undefined || yVal === undefined) continue
    const x: number = xVal
    const y: number = yVal
    if (Number.isNaN(x) || Number.isNaN(y)) continue

    const pos = worldToScreen(x, y)

    if (
      pos.x + pos.size < 0 ||
      pos.y + pos.size < 0 ||
      pos.x > canvas.width ||
      pos.y > canvas.height
    ) continue

    ctx.fillStyle = color
    ctx.fillRect(pos.x, pos.y, pos.size, pos.size)
  }
}
