import type { Pixel } from "./model"

export const state = {
  message: "",
  pending: null as {
    requestId: string
    x: number
    y: number
    color: string
  } | null,
  hoverPixel: null as Pixel | null,
  selectedPixel: null as Pixel | null,
  selectedColor: "#ff0000"
}

export type ConnectionStatus =
  | "connecting"
  | "handshaking"
  | "syncing"
  | "connected"
  | "reconnecting"
  | "disconnected"

export const connection = {
  status: "connecting" as ConnectionStatus
}
