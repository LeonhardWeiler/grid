import config from "../../internal/protocol/config.json"
export { config }
export interface PixelEvent { version: number; x: number; y: number; color: string }
export type ServerMessage =
  | { type: "init"; version: number; pixels?: Record<string, string>; cooldownMs: number; protocol?: number; gridSize?: number }
  | { type: "sync"; version: number; events?: PixelEvent[]; cooldownMs: number }
  | ({ type: "pixel_update" } & PixelEvent)
  | { type: "cooldown"; cooldownMs: number; end?: number }
