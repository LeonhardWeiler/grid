import { ctx, canvas, view } from "../dom"
import { camera, BASE_PIXEL_SIZE, GRID_SIZE } from "../camera/camera"
import { raster, flushRaster } from "./raster"

export function drawPixels(): void {
  flushRaster()
  const scale = BASE_PIXEL_SIZE * camera.zoom
  ctx.imageSmoothingEnabled = false
  ctx.drawImage(
    raster,
    (0 - camera.x) * scale + view.width / 2,
    (0 - camera.y) * scale + view.height / 2,
    GRID_SIZE * scale,
    GRID_SIZE * scale
  )
}
