import {minimap,minimapCtx,canvas} from "../dom"
import {GRID_SIZE,camera,BASE_PIXEL_SIZE} from "../camera/camera"
import {raster,flushRaster} from "./raster"

minimap.width=minimap.height=180
export function drawMinimap():void {
  flushRaster()
  const width=minimap.width,height=minimap.height
  minimapCtx.imageSmoothingEnabled=false
  minimapCtx.drawImage(raster,0,0,width,height)
  const scale=width/GRID_SIZE
  const visibleWidth=canvas.width/(BASE_PIXEL_SIZE*camera.zoom)
  const visibleHeight=canvas.height/(BASE_PIXEL_SIZE*camera.zoom)
  minimapCtx.strokeStyle="white"
  minimapCtx.lineWidth=1
  minimapCtx.strokeRect((camera.x-visibleWidth/2)*scale,(camera.y-visibleHeight/2)*scale,visibleWidth*scale,visibleHeight*scale)
}
