import config from "../../internal/protocol/config.json"
export { config }
export interface PixelEvent { version: number; x: number; y: number; color: string }
export type ServerMessage =
  | { type: "accepted"; requestId: string; version: number; cooldownMs: number }
  | { type: "rejected"; requestId: string; code: string; cooldownMs: number }
  | { type: "error"; code: string; cooldownMs: number }
  | { type: "init"; version: number; pixels?: Record<string, string>; cooldownMs: number; protocol?: number; gridSize?: number }
  | { type: "sync"; version: number; events?: PixelEvent[]; cooldownMs: number }
  | ({ type: "pixel_update" } & PixelEvent)
  | { type: "cooldown"; cooldownMs: number; end?: number }
