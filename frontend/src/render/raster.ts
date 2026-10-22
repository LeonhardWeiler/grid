import { config } from "../protocol"
import { onPixelChange } from "../model"

const size = config.gridSize
export const raster = document.createElement("canvas")
raster.width = raster.height = size
const context = raster.getContext("2d")!
const image = context.createImageData(size, size)
const colors = new Map(
  config.colors.map((c) => [
    c.value,
    [
      parseInt(c.value.slice(1, 3), 16),
      parseInt(c.value.slice(3, 5), 16),
      parseInt(c.value.slice(5, 7), 16)
    ] as const
  ])
)
let dirty = true
let minX = 0,
  minY = 0,
  maxX = size - 1,
  maxY = size - 1
function reset() {
  for (let i = 0; i < image.data.length; i += 4) {
    image.data[i] = 17
    image.data[i + 1] = 17
    image.data[i + 2] = 17
    image.data[i + 3] = 255
  }
  minX = minY = 0
  maxX = maxY = size - 1
  dirty = true
}
reset()
onPixelChange((key, color) => {
  if (key === null) {
    reset()
    return
  }
  const split = key.indexOf(":")
  const x = Number(key.slice(0, split)),
    y = Number(key.slice(split + 1))
  const rgb = colors.get(color!)
  if (
    !rgb ||
    !Number.isInteger(x) ||
    !Number.isInteger(y) ||
    x < 0 ||
    y < 0 ||
    x >= size ||
    y >= size
  )
    return
  const index = (y * size + x) * 4
  image.data[index] = rgb[0]
  image.data[index + 1] = rgb[1]
  image.data[index + 2] = rgb[2]
  if (!dirty) {
    minX = maxX = x
    minY = maxY = y
  } else {
    minX = Math.min(minX, x)
    minY = Math.min(minY, y)
    maxX = Math.max(maxX, x)
    maxY = Math.max(maxY, y)
  }
  dirty = true
})
export function flushRaster(): void {
  if (!dirty) return
  context.putImageData(
    image,
    0,
    0,
    minX,
    minY,
    maxX - minX + 1,
    maxY - minY + 1
  )
  dirty = false
}
