import { canvas, coordsEl,view,resizeCanvas } from "../dom"
import { toCenteredCoords } from "../utils/gridCoords"

import {
  camera,
  BASE_PIXEL_SIZE,
  GRID_SIZE,
  clampCamera
} from "./camera"

import { screenToWorld } from "../utils/coords"
import { requestRender } from "../render/render"
import { state } from "../state"
import type { Pixel } from "../state"
import {exceedsDragThreshold} from "./gesture"

let dragging = false
let moved = false
const start={x:0,y:0}

const last: { x: number; y: number } = {
  x: 0,
  y: 0
}

export function setupControls(): void {

  canvas.addEventListener("mousedown", (e: MouseEvent) => {
    dragging = true
    moved = false

    last.x = e.clientX
    last.y = e.clientY
    start.x=e.clientX;start.y=e.clientY
  })

  window.addEventListener("mouseup", () => {
    dragging = false
  })

  window.addEventListener("mousemove", (e: MouseEvent) => {
    if (dragging) {
      const dx = e.clientX - last.x
      const dy = e.clientY - last.y

      if (exceedsDragThreshold(start,{x:e.clientX,y:e.clientY})) {
        moved = true
      }

      camera.tx -= dx / (BASE_PIXEL_SIZE * camera.tzoom)
      camera.ty -= dy / (BASE_PIXEL_SIZE * camera.tzoom)

      clampCamera()

      last.x = e.clientX
      last.y = e.clientY
    }

    const world: Pixel = screenToWorld(e.clientX, e.clientY)

    if (
      world.x < 0 || world.x >= GRID_SIZE ||
      world.y < 0 || world.y >= GRID_SIZE
    ) {
      if (state.hoverPixel !== null) {
        state.hoverPixel = null
        requestRender()
      }
      return
    }

    state.hoverPixel = world

    const centered = toCenteredCoords(world.x, world.y)
    coordsEl.textContent = `${centered.x} | ${centered.y}`

    requestRender()
  })

  window.addEventListener("wheel", (e: WheelEvent) => {
    e.preventDefault()

    const factor = 1.1
    camera.tzoom *= e.deltaY < 0 ? factor : 1 / factor

    clampCamera()
  }, { passive: false })

  window.addEventListener("resize", () => {
    resizeCanvas()
    clampCamera()

    requestRender()
  })
}

export function didDrag(): boolean {
  return moved
}
