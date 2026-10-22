const canvasEl = document.getElementById("canvas")
if (!(canvasEl instanceof HTMLCanvasElement)) {
  throw new Error("Canvas element not found")
}
export const canvas: HTMLCanvasElement = canvasEl

const ctxEl = canvas.getContext("2d")
if (!ctxEl) {
  throw new Error("2D context not available")
}
export const ctx: CanvasRenderingContext2D = ctxEl

export const view = { width:window.innerWidth,height:window.innerHeight,dpr:1 }

const minimapEl = document.getElementById("minimap")
if (!(minimapEl instanceof HTMLCanvasElement)) {
  throw new Error("Minimap element not found")
}
export const minimap: HTMLCanvasElement = minimapEl

const minimapCtxEl = minimap.getContext("2d")
if (!minimapCtxEl) {
  throw new Error("Minimap 2D context not available")
}
export const minimapCtx: CanvasRenderingContext2D = minimapCtxEl

export function resizeCanvas():void {
  const rect=canvas.getBoundingClientRect()
  view.width=rect.width;view.height=rect.height;view.dpr=Math.min(3,window.devicePixelRatio||1)
  canvas.width=Math.round(view.width*view.dpr);canvas.height=Math.round(view.height*view.dpr)
  ctx.setTransform(view.dpr,0,0,view.dpr,0,0)
  minimap.width=Math.round(minimap.clientWidth*view.dpr);minimap.height=Math.round(minimap.clientHeight*view.dpr)
  minimapCtx.setTransform(view.dpr,0,0,view.dpr,0,0)
}
resizeCanvas()

const cooldownElRaw = document.getElementById("cooldown")
if (!(cooldownElRaw instanceof HTMLButtonElement)) {
  throw new Error("Cooldown element not found")
}
export const cooldownEl: HTMLButtonElement = cooldownElRaw

const coordsElRaw = document.getElementById("coords")
if (!(coordsElRaw instanceof HTMLElement)) {
  throw new Error("Coords element not found")
}
export const coordsEl: HTMLElement = coordsElRaw

const paletteElRaw = document.getElementById("palette")
if (!(paletteElRaw instanceof HTMLElement)) {
  throw new Error("Palette element not found")
}
export const paletteEl: HTMLElement = paletteElRaw

const connectionOverlayRaw = document.getElementById("connection-overlay")
if (!(connectionOverlayRaw instanceof HTMLElement)) {
  throw new Error("Connection overlay element not found")
}
export const connectionOverlay: HTMLElement = connectionOverlayRaw

const connectionWarningRaw = document.getElementById("connection-warning")
if (!(connectionWarningRaw instanceof HTMLElement)) {
  throw new Error("Connection warning element not found")
}
export const connectionWarning: HTMLElement = connectionWarningRaw
