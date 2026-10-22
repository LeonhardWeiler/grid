import { pixels } from "./model"
import { requestRender } from "./render/render"
import { setCooldown } from "./ui/cooldown"
import { connection, state } from "./state"

import type { ServerMessage } from "./protocol"

export function createWS(clientId: string) {
  let ws: WebSocket | null = null
  // Only an in-memory version corresponds to the in-memory canvas.
  let version = 0
  let reconnectDelay = 250
  let reconnectTimer: number | null = null
  let handshakeTimer: number | null = null
  let lastMessage = Date.now()
  let pendingTimer: number | null = null

  function finishPending(message: string, accepted = false) {
    if (pendingTimer !== null) window.clearTimeout(pendingTimer)
    pendingTimer = null
    state.pending = null
    if (accepted) state.selectedPixel = null
    state.message = message
    requestRender()
  }

  function clearHandshake() {
    if (handshakeTimer !== null) window.clearTimeout(handshakeTimer)
    handshakeTimer = null
  }

  function connect() {
    if (!navigator.onLine) {
      connection.status = "disconnected"
      return
    }
    if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) return
    if (reconnectTimer !== null) window.clearTimeout(reconnectTimer)
    reconnectTimer = null
    const url = new URL("/ws", window.location.href)
    url.protocol = location.protocol === "https:" ? "wss:" : "ws:"
    const socket = new WebSocket(url)
    ws = socket
    connection.status = "connecting"
    handshakeTimer = window.setTimeout(() => socket.close(), 10000)

    socket.onopen = () => {
      connection.status = "handshaking"
      lastMessage = Date.now()
      socket.send(JSON.stringify({ type: "init_client", clientId, lastVersion: version }))
    }

    socket.onmessage = (e: MessageEvent) => {
      if (ws !== socket) return
      lastMessage = Date.now()
      let msg: ServerMessage
      try { msg = JSON.parse(e.data) } catch { return }
      switch (msg.type) {
        case "accepted":
          if (state.pending?.requestId === msg.requestId) finishPending("Pixel saved.", true)
          setCooldown(msg.cooldownMs)
          break
        case "rejected":
          if (state.pending?.requestId === msg.requestId) {
            finishPending(msg.code === "unavailable" ? "Storage unavailable. Your selection is kept." : msg.code === "invalid_pixel" ? "This pixel is invalid. Select another pixel." : "Please wait before trying again. Your selection is kept.")
            if (msg.cooldownMs > 0) setCooldown(msg.cooldownMs)
          }
          break
        case "error":
          state.message = "Sync is busy. Retrying shortly."
          break
        case "init":
          pixels.clear()
          for (const [key, color] of Object.entries(msg.pixels ?? {})) pixels.set(key, color)
          version = msg.version ?? 0
          setCooldown(msg.cooldownMs)
          clearHandshake()
          connection.status = "connected"
          reconnectDelay = 250
          requestRender()
          break
        case "sync":
          for (const event of msg.events ?? []) {
            if (event.version > version) pixels.set(`${event.x}:${event.y}`, event.color)
          }
          version = msg.version ?? version
          setCooldown(msg.cooldownMs)
          requestRender()
          break
        case "pixel_update":
          if (msg.version === undefined || msg.version <= version) break
          if (msg.version !== version + 1) {
            socket.send(JSON.stringify({ type: "sync", lastVersion: version }))
            break
          }
          if (msg.x !== undefined && msg.y !== undefined && msg.color) {
            pixels.set(`${msg.x}:${msg.y}`, msg.color)
            version = msg.version
            requestRender()
          }
          break
        case "cooldown":
          setCooldown(msg.cooldownMs)
          break
      }
    }

    socket.onclose = () => {
      if (ws !== socket) return
      clearHandshake()
      ws = null
      if (state.pending) finishPending("Connection lost. Your selection is kept; check the canvas before retrying.")
      connection.status = navigator.onLine ? "reconnecting" : "disconnected"
      if (!navigator.onLine) return
      reconnectTimer = window.setTimeout(connect, reconnectDelay + Math.random() * 250)
      reconnectDelay = Math.min(reconnectDelay * 2, 15000)
    }
    socket.onerror = () => socket.close()
  }

  function resume() {
    if (!navigator.onLine) return
    if (ws?.readyState === WebSocket.OPEN && Date.now() - lastMessage > 45000) ws.close()
    if (!ws || ws.readyState === WebSocket.CLOSED) connect()
  }
  window.addEventListener("online", resume)
  window.addEventListener("offline", () => {
    connection.status = "disconnected"
    ws?.close()
  })
  window.addEventListener("pageshow", resume)
  document.addEventListener("visibilitychange", () => { if (!document.hidden) resume() })
  // Application-level heartbeat also detects a half-open connection in browsers.
  window.setInterval(() => {
    if (ws?.readyState !== WebSocket.OPEN || connection.status !== "connected") return
    if (Date.now() - lastMessage > 45000) { ws.close(); return }
    ws.send(JSON.stringify({ type: "sync", lastVersion: version }))
  }, 20000)
  connect()

  return {
    placePixel(x: number, y: number, color: string) {
      if (ws?.readyState !== WebSocket.OPEN || connection.status !== "connected" || state.pending) return false
      const requestId = crypto.randomUUID()
      state.pending = { requestId, x, y, color }
      state.message = "Saving pixel…"
      ws.send(JSON.stringify({ type:"set_pixel", x, y, color, requestId }))
      pendingTimer = window.setTimeout(() => {
        finishPending("No confirmation received. Your selection is kept; check the canvas before retrying.")
        ws?.close()
      },10000)
      return true
    }
  }
}
