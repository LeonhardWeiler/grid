import {canvas,cooldownEl,coordsEl,view} from "./dom"
import {state,connection} from "./state"
import type {Pixel} from "./model"
import {updateConnectionUI} from "./ui/connection"
import {setupPalette} from "./ui/palette"
import {GRID_SIZE,fitToScreen,camera,clampCamera} from "./camera/camera"
import {setupControls,didDrag,zoomCamera} from "./camera/controls"
import {render,requestRender} from "./render/render"
import {updateButtonUI} from "./ui/button"
import {isCooldownActive} from "./ui/cooldown"
import {screenToWorld} from "./utils/coords"
import {createWS} from "./ws"
import {browserIdentity} from "./identity"

const identity=browserIdentity()
if(!identity.persistent)state.message="Browser storage is unavailable. Your identity lasts for this page only."
else if(identity.repaired)state.message="Your browser identity was repaired."
else state.message="Choose a color, select a pixel, then Place. Drag to pan; scroll or pinch to zoom."
const ws=createWS(identity.id)
const xInput=document.getElementById("pixel-x") as HTMLInputElement
const yInput=document.getElementById("pixel-y") as HTMLInputElement
for(const input of [xInput,yInput]){input.max=String(GRID_SIZE-1);input.value=String(Math.floor(GRID_SIZE/2))}

function selectPixel(pixel:Pixel,center=false){
  if(!Number.isInteger(pixel.x)||!Number.isInteger(pixel.y)||pixel.x<0||pixel.y<0||pixel.x>=GRID_SIZE||pixel.y>=GRID_SIZE)return
  state.selectedPixel=pixel
  xInput.value=String(pixel.x);yInput.value=String(pixel.y)
  coordsEl.textContent=pixel.x+" | "+pixel.y
  if(center){camera.tx=pixel.x+.5;camera.ty=pixel.y+.5;clampCamera()}
  requestRender()
}
function confirmPixel(){
  if(connection.status!=="connected"||!state.selectedPixel||state.pending||isCooldownActive())return
  ws.placePixel(state.selectedPixel.x,state.selectedPixel.y,state.selectedColor)
  requestRender()
}
canvas.addEventListener("click",event=>{
  if(didDrag()||event.button!==0)return
  selectPixel(screenToWorld(event.clientX,event.clientY))
})
document.getElementById("coordinates")!.addEventListener("submit",event=>{
  event.preventDefault()
  selectPixel({x:xInput.valueAsNumber,y:yInput.valueAsNumber},true)
})
cooldownEl.addEventListener("click",confirmPixel)
canvas.addEventListener("keydown",event=>{
  if(event.ctrlKey||event.metaKey||event.altKey)return
  const step=event.shiftKey?10:1
  const pixel=state.selectedPixel ?? {x:Math.floor(camera.x),y:Math.floor(camera.y)}
  let next:Pixel|null=null
  switch(event.code){
    case "ArrowLeft":next={x:Math.max(0,pixel.x-step),y:pixel.y};break
    case "ArrowRight":next={x:Math.min(GRID_SIZE-1,pixel.x+step),y:pixel.y};break
    case "ArrowUp":next={x:pixel.x,y:Math.max(0,pixel.y-step)};break
    case "ArrowDown":next={x:pixel.x,y:Math.min(GRID_SIZE-1,pixel.y+step)};break
    case "Enter":case "Space":event.preventDefault();confirmPixel();return
    default:return
  }
  event.preventDefault()
  selectPixel(next,true)
})
const help=document.getElementById("help") as HTMLDialogElement
document.getElementById("help-open")!.addEventListener("click",()=>help.showModal())
document.getElementById("zoom-in")!.addEventListener("click",()=>zoomCamera(2))
document.getElementById("zoom-out")!.addEventListener("click",()=>zoomCamera(.5))
document.getElementById("reset-view")!.addEventListener("click",()=>{fitToScreen();requestRender()})

setupControls()
setupPalette()
fitToScreen()
function loop(){
  const oldX=camera.x,oldY=camera.y,oldZoom=camera.zoom
  camera.x+=(camera.tx-camera.x)*.15
  camera.y+=(camera.ty-camera.y)*.15
  camera.zoom+=(camera.tzoom-camera.zoom)*.15
  clampCamera()
  if(Math.abs(oldX-camera.x)>.001||Math.abs(oldY-camera.y)>.001||Math.abs(oldZoom-camera.zoom)>.001)requestRender()
  render()
  updateButtonUI()
  updateConnectionUI()
  requestAnimationFrame(loop)
}
loop()
