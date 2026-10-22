import { canvas, minimap, coordsEl, view, resizeCanvas } from "../dom"
import { camera, BASE_PIXEL_SIZE, GRID_SIZE, clampCamera } from "./camera"
import { screenToWorld } from "../utils/coords"
import { requestRender } from "../render/render"
import { state } from "../state"
import { exceedsDragThreshold } from "./gesture"

type Pointer = { x: number; y: number; start: { x: number; y: number } }
const pointers = new Map<number, Pointer>()
let moved = false

export function zoomCamera(
  factor: number,
  clientX?: number,
  clientY?: number
): void {
  const rect = canvas.getBoundingClientRect()
  const x = (clientX ?? rect.left + view.width / 2) - rect.left - view.width / 2
  const y = (clientY ?? rect.top + view.height / 2) - rect.top - view.height / 2
  const anchorX = camera.tx + x / (BASE_PIXEL_SIZE * camera.tzoom)
  const anchorY = camera.ty + y / (BASE_PIXEL_SIZE * camera.tzoom)
  camera.tzoom *= factor
  clampCamera()
  camera.tx = anchorX - x / (BASE_PIXEL_SIZE * camera.tzoom)
  camera.ty = anchorY - y / (BASE_PIXEL_SIZE * camera.tzoom)
  clampCamera()
  requestRender()
}

export function setupControls(): void {
  canvas.addEventListener("pointerdown", (event) => {
    if (event.pointerType === "mouse" && event.button !== 0) return
    if (pointers.size === 0) moved = false
    pointers.set(event.pointerId, {
      x: event.clientX,
      y: event.clientY,
      start: { x: event.clientX, y: event.clientY }
    })
    if (pointers.size > 1) moved = true
    canvas.setPointerCapture(event.pointerId)
    canvas.focus({ preventScroll: true })
  })
  canvas.addEventListener("pointermove", (event) => {
    const point = pointers.get(event.pointerId)
    if (point) {
      const previous = [...pointers.values()].map((p) => ({ x: p.x, y: p.y }))
      const dx = event.clientX - point.x,
        dy = event.clientY - point.y
      point.x = event.clientX
      point.y = event.clientY
      moved ||= exceedsDragThreshold(point.start, point)
      if (pointers.size === 2) {
        moved = true
        const current = [...pointers.values()]
        const a = previous[0]!,
          b = previous[1]!,
          c = current[0]!,
          d = current[1]!
        const before = Math.hypot(a.x - b.x, a.y - b.y)
        const after = Math.hypot(c.x - d.x, c.y - d.y)
        if (before > 0 && after > 0)
          zoomCamera(after / before, (a.x + b.x) / 2, (a.y + b.y) / 2)
        camera.tx -=
          (c.x + d.x - a.x - b.x) / 2 / (BASE_PIXEL_SIZE * camera.tzoom)
        camera.ty -=
          (c.y + d.y - a.y - b.y) / 2 / (BASE_PIXEL_SIZE * camera.tzoom)
      } else if (pointers.size === 1 && moved) {
        camera.tx -= dx / (BASE_PIXEL_SIZE * camera.tzoom)
        camera.ty -= dy / (BASE_PIXEL_SIZE * camera.tzoom)
      }
      clampCamera()
    }
    const pixel = screenToWorld(event.clientX, event.clientY)
    if (
      pixel.x >= 0 &&
      pixel.y >= 0 &&
      pixel.x < GRID_SIZE &&
      pixel.y < GRID_SIZE
    ) {
      state.hoverPixel = pixel
      coordsEl.textContent = pixel.x + " | " + pixel.y
    } else state.hoverPixel = null
    requestRender()
  })
  canvas.addEventListener("pointerup", (event) =>
    pointers.delete(event.pointerId)
  )
  const cancel = (event: PointerEvent) => {
    if (pointers.delete(event.pointerId)) moved = true
  }
  canvas.addEventListener("pointercancel", cancel)
  canvas.addEventListener("lostpointercapture", cancel)
  canvas.addEventListener("pointerleave", () => {
    if (pointers.size === 0) {
      state.hoverPixel = null
      requestRender()
    }
  })
  canvas.addEventListener(
    "wheel",
    (event) => {
      if (event.ctrlKey || event.metaKey) return
      event.preventDefault()
      zoomCamera(
        Math.exp(-Math.max(-300, Math.min(300, event.deltaY)) * 0.002),
        event.clientX,
        event.clientY
      )
    },
    { passive: false }
  )
  const resize = () => {
    resizeCanvas()
    clampCamera()
    requestRender()
  }
  window.addEventListener("resize", resize)
  const observer = new ResizeObserver(resize)
  observer.observe(canvas)
  observer.observe(minimap)
}
export function didDrag(): boolean {
  return moved
}
