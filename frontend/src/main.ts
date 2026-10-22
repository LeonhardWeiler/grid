import { canvas, cooldownEl } from "./dom"
import { state, connection } from "./state"
import { updateConnectionUI } from "./ui/connection"
import { setupPalette } from "./ui/palette"
import { GRID_SIZE, fitToScreen, camera, clampCamera} from "./camera/camera"
import { setupControls, didDrag} from "./camera/controls"
import { render, requestRender} from "./render/render"
import { updateButtonUI } from "./ui/button"
import { isCooldownActive } from "./ui/cooldown"
import { screenToWorld } from "./utils/coords"
import { createWS } from "./ws"

let clientId = localStorage.getItem("clientId")

if (!clientId) {
  clientId = crypto.randomUUID()
  localStorage.setItem(
    "clientId",
    clientId
  )
}

const ws = createWS(clientId)

canvas.addEventListener("click", (e) => {
  if (didDrag()) return

  const world =
    screenToWorld(
      e.clientX,
      e.clientY
    )

  if (
    world.x < 0 ||
    world.x >= GRID_SIZE ||
    world.y < 0 ||
    world.y >= GRID_SIZE
  ) return

  if (
    state.selectedPixel &&
    state.selectedPixel.x === world.x &&
    state.selectedPixel.y === world.y
  ) {
    confirmPixel()
    return
  }

  state.selectedPixel = world
  updateButtonUI()
  requestRender()
})

function confirmPixel() {
  if (connection.status !== "connected") return
  if (!state.selectedPixel) return
  if (isCooldownActive()) return

  ws.send(JSON.stringify({
    type: "set_pixel",
    x: state.selectedPixel.x,
    y: state.selectedPixel.y,
    color: state.selectedColor
  }))

  state.selectedPixel = null
  updateButtonUI()
  requestRender()
}

cooldownEl.addEventListener(
  "click",
  confirmPixel
)

window.addEventListener(
  "keydown",
  (e) => {
    if (e.code !== "Space") return
    e.preventDefault()
    confirmPixel()
  }
)

setupControls()
setupPalette()
fitToScreen()

function loop() {
  const oldX = camera.x
  const oldY = camera.y
  const oldZoom = camera.zoom

  camera.x += (camera.tx - camera.x) * 0.15
  camera.y += (camera.ty - camera.y) * 0.15
  camera.zoom += (camera.tzoom - camera.zoom) * 0.15
  clampCamera()

  const moved =
    Math.abs(oldX - camera.x) > 0.001 ||
    Math.abs(oldY - camera.y) > 0.001 ||
    Math.abs(oldZoom - camera.zoom) > 0.001

  if (moved) {
    requestRender()
  }

  render()
  updateButtonUI()
  requestAnimationFrame(loop)
  updateConnectionUI()
}

loop()

