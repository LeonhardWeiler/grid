import type {PixelEvent} from "./protocol"

export function applyEvents(pixels: Map<string,string>, version: number, events: PixelEvent[], target: number): {version:number;changed:boolean} {
  if (!Number.isSafeInteger(target) || target<0) throw new Error("Invalid version")
  if (target<version) return {version,changed:false}
  const missing=events.filter(event=>event.version>version)
  if (target-version!==missing.length || missing.some((event,i)=>event.version!==version+i+1)) throw new Error("Noncontiguous delta")
  for (const event of missing) pixels.set(event.x+":"+event.y,event.color)
  return {version:target,changed:missing.length>0}
}
