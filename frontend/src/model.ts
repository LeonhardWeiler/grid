export interface Pixel { x: number; y: number }
type PixelChange = (key: string | null, color?: string) => void
const listeners = new Set<PixelChange>()
export function onPixelChange(listener: PixelChange): void { listeners.add(listener) }
class PixelMap extends Map<string,string> {
  revision=0
  override set(key:string,color:string):this {
    if(this.get(key)===color)return this
    super.set(key,color);this.revision++
    for(const listener of listeners)listener(key,color)
    return this
  }
  override clear():void {
    super.clear();this.revision++
    for(const listener of listeners)listener(null)
  }
}
export const pixels = new PixelMap()
