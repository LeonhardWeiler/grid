import { canvas, view } from "../dom"
import { camera, BASE_PIXEL_SIZE } from "../camera/camera"

import type { Pixel } from "../model"

export interface ScreenCoords {
  x: number
  y: number
  size: number
}

export function screenToWorld(clientX: number, clientY: number): Pixel {
  if (!canvas) throw new Error("Canvas not found")
  const rect = canvas.getBoundingClientRect()
  clientX -= rect.left
  clientY -= rect.top

  const x = Math.floor(
    (clientX - view.width / 2) / (BASE_PIXEL_SIZE * camera.zoom) + camera.x
  )

  const y = Math.floor(
    (clientY - view.height / 2) / (BASE_PIXEL_SIZE * camera.zoom) + camera.y
  )

  return { x, y }
}

export function worldToScreen(x: number, y: number): ScreenCoords {
  if (!canvas) throw new Error("Canvas not found")

  const size = BASE_PIXEL_SIZE * camera.zoom
  return {
    x: (x - camera.x) * size + view.width / 2,
    y: (y - camera.y) * size + view.height / 2,
    size
  }
}
