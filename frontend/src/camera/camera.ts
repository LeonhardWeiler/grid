import { canvas } from "../dom"

export const GRID_SIZE: number = 512
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
  camera.zoom = 0.6
  camera.tzoom = 0.6
}

export function clampCamera(): void {
  const visibleWidth = canvas.width / (BASE_PIXEL_SIZE * camera.tzoom)
  const visibleHeight = canvas.height / (BASE_PIXEL_SIZE * camera.tzoom)

  const halfW = visibleWidth / 2
  const halfH = visibleHeight / 2

  camera.tx = Math.max(halfW, Math.min(GRID_SIZE - halfW, camera.tx))
  camera.ty = Math.max(halfH, Math.min(GRID_SIZE - halfH, camera.ty))
}
