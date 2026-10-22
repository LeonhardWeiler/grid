import { minimap, minimapCtx, canvas } from "../dom"
import { GRID_SIZE, camera, BASE_PIXEL_SIZE } from "../camera/camera"
import { pixels } from "../model"

const MINIMAP_SIZE: number = 180

minimap.width = MINIMAP_SIZE
minimap.height = MINIMAP_SIZE

export function drawMinimap(): void {
  minimapCtx.clearRect(0, 0, MINIMAP_SIZE, MINIMAP_SIZE)
  minimapCtx.fillStyle = "#111"
  minimapCtx.fillRect(0, 0, MINIMAP_SIZE, MINIMAP_SIZE)

  const scale: number = MINIMAP_SIZE / GRID_SIZE

  for (const [key, color] of pixels) {
    const parts = key.split(":").map(Number)

    if (parts.length !== 2) continue
    const xVal = parts[0]
    const yVal = parts[1]
    if (xVal === undefined || yVal === undefined) continue
    const x: number = xVal
    const y: number = yVal
    if (Number.isNaN(x) || Number.isNaN(y)) continue

    minimapCtx.fillStyle = color
    minimapCtx.fillRect(x * scale, y * scale, Math.max(1, scale), Math.max(1, scale))
  }

  const visibleWidth: number = canvas.width / (BASE_PIXEL_SIZE * camera.zoom)
  const visibleHeight: number = canvas.height / (BASE_PIXEL_SIZE * camera.zoom)

  const viewX: number = (camera.x - visibleWidth / 2) * scale
  const viewY: number = (camera.y - visibleHeight / 2) * scale
  const viewW: number = visibleWidth * scale
  const viewH: number = visibleHeight * scale

  minimapCtx.strokeStyle = "white"
  minimapCtx.lineWidth = 1
  minimapCtx.strokeRect(viewX, viewY, viewW, viewH)
}
