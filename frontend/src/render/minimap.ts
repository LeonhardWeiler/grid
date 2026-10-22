import {minimap,minimapCtx,canvas,view} from "../dom"
import {GRID_SIZE,camera,BASE_PIXEL_SIZE} from "../camera/camera"
import {raster,flushRaster} from "./raster"

export function drawMinimap():void {
  flushRaster()
  const width=minimap.clientWidth,height=minimap.clientHeight
  minimapCtx.imageSmoothingEnabled=false
  minimapCtx.drawImage(raster,0,0,width,height)
  const scale=width/GRID_SIZE
  const visibleWidth=view.width/(BASE_PIXEL_SIZE*camera.zoom)
  const visibleHeight=view.height/(BASE_PIXEL_SIZE*camera.zoom)
  minimapCtx.strokeStyle="white"
  minimapCtx.lineWidth=1
  minimapCtx.strokeRect((camera.x-visibleWidth/2)*scale,(camera.y-visibleHeight/2)*scale,visibleWidth*scale,visibleHeight*scale)
}
