import { canvas ,view} from "../dom"
import { clampView } from "./bounds"

import { config } from "../protocol"
export const GRID_SIZE: number = config.gridSize
export const BASE_PIXEL_SIZE: number = 40

export interface Camera {
  x: number
  y: number
  tx: number
  ty: number
  zoom: number
  tzoom: number
}

export const camera: Camera = {
  x: GRID_SIZE / 2,
  y: GRID_SIZE / 2,
  tx: GRID_SIZE / 2,
  ty: GRID_SIZE / 2,
  zoom: 0.6,
  tzoom: 0.6
}

export function fitToScreen(): void {
  camera.x = GRID_SIZE / 2
  camera.y = GRID_SIZE / 2
  camera.tx = GRID_SIZE / 2
  camera.ty = GRID_SIZE / 2
  camera.zoom = Math.max(view.width,view.height)/(GRID_SIZE*BASE_PIXEL_SIZE)
  camera.tzoom = camera.zoom
  clampCamera()
}

export function clampCamera(): void {
  const target = clampView(
    { x: camera.tx, y: camera.ty, zoom: camera.tzoom },
    view.width, view.height, GRID_SIZE, BASE_PIXEL_SIZE
  )
  camera.tx = target.x
  camera.ty = target.y
  camera.tzoom = target.zoom
  Object.assign(camera, clampView(camera, view.width, view.height, GRID_SIZE, BASE_PIXEL_SIZE))
}
