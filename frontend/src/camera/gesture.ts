export function exceedsDragThreshold(
  start: { x: number; y: number },
  point: { x: number; y: number }
): boolean {
  return Math.hypot(point.x - start.x, point.y - start.y) > 4
}
