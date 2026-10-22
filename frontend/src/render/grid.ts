import { canvas, ctx, view } from "../dom"
import { camera, BASE_PIXEL_SIZE, GRID_SIZE } from "../camera/camera"

export function drawGrid(): void {
  const baseStep: number = BASE_PIXEL_SIZE * camera.zoom

  let cellStep: number = 1
  let step: number = baseStep

  while (step < 12) {
    cellStep *= 2
    step = baseStep * cellStep
  }

  ctx.strokeStyle = "rgba(255,255,255,0.15)"
  ctx.lineWidth = 1

  const visibleWidth: number = view.width / baseStep
  const visibleHeight: number = view.height / baseStep

  const startWorldX: number = Math.max(
    0,
    Math.floor(camera.x - visibleWidth / 2)
  )
  const endWorldX: number = Math.min(
    GRID_SIZE,
    Math.ceil(camera.x + visibleWidth / 2)
  )

  const startWorldY: number = Math.max(
    0,
    Math.floor(camera.y - visibleHeight / 2)
  )
  const endWorldY: number = Math.min(
    GRID_SIZE,
    Math.ceil(camera.y + visibleHeight / 2)
  )

  for (
    let x: number = Math.floor(startWorldX / cellStep) * cellStep;
    x <= endWorldX;
    x += cellStep
  ) {
    const screenX: number = (x - camera.x) * baseStep + view.width / 2

    ctx.beginPath()
    ctx.moveTo(screenX, 0)
    ctx.lineTo(screenX, view.height)
    ctx.stroke()
  }

  for (
    let y: number = Math.floor(startWorldY / cellStep) * cellStep;
    y <= endWorldY;
    y += cellStep
  ) {
    const screenY: number = (y - camera.y) * baseStep + view.height / 2

    ctx.beginPath()
    ctx.moveTo(0, screenY)
    ctx.lineTo(view.width, screenY)
    ctx.stroke()
  }
}
