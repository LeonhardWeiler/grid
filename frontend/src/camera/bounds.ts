interface View {
  x: number
  y: number
  zoom: number
}

export function clampView(view: View, width: number, height: number, gridSize: number, pixelSize: number): View {
  // The entire viewport must fit within the grid, including on wide displays.
  const minZoom = Math.max(width, height) / (gridSize * pixelSize)
  const zoom = Math.max(minZoom, Math.min(2, view.zoom))
  const halfWidth = Math.min(gridSize / 2, width / (2 * pixelSize * zoom))
  const halfHeight = Math.min(gridSize / 2, height / (2 * pixelSize * zoom))
  return {
    x: Math.max(halfWidth, Math.min(gridSize - halfWidth, view.x)),
    y: Math.max(halfHeight, Math.min(gridSize - halfHeight, view.y)),
    zoom
  }
}
