export interface Pixel { x: number; y: number }
// The authoritative local view belongs to the model, not the renderer.
export const pixels = new Map<string, string>()
