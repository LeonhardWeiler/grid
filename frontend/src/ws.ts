import { pixels } from "./model"
import { requestRender } from "./render/render"
import { setCooldown } from "./ui/cooldown"
import { connection, state } from "./state"
import { config } from "./protocol"
import type { ServerMessage, PixelEvent } from "./protocol"
import { applyEvents } from "./sync"
import {requestFrame} from "./frame"
import type {ConnectionStatus} from "./state"
function setStatus(status:ConnectionStatus){connection.status=status;requestFrame()}

function parseMessage(data: string): ServerMessage {
  const msg = JSON.parse(data)
  if (!msg || typeof msg !== "object") throw new Error("Invalid server message")
  const validVersion = (v: unknown) => Number.isSafeInteger(v) && (v as number) >= 0
  const validCooldown = (v: unknown) => typeof v === "number" && Number.isFinite(v) && v >= 0 && v <= 60000
  const validPixel = (e: PixelEvent) => e && Number.isInteger(e.x) && Number.isInteger(e.y) && e.x >= 0 && e.y >= 0 && e.x < config.gridSize && e.y < config.gridSize && config.colors.some(c => c.value === e.color)
  switch (msg.type) {
    case "init":
      if (!validVersion(msg.version) || !validCooldown(msg.cooldownMs) || (msg.protocol !== undefined && msg.protocol !== config.protocol) || (msg.gridSize !== undefined && msg.gridSize !== config.gridSize)) throw new Error("Incompatible canvas; reload the page")
      if (msg.pixels !== undefined && (!msg.pixels || typeof msg.pixels !== "object" || Array.isArray(msg.pixels))) throw new Error("Invalid snapshot")
      if (Object.keys(msg.pixels ?? {}).length > config.gridSize ** 2) throw new Error("Oversized snapshot")
      for (const [key, color] of Object.entries(msg.pixels ?? {})) {
        const [x,y] = key.split(":").map(Number)
        if (key !== x + ":" + y || !validPixel({x:x!, y:y!, color:color as string, version:0})) throw new Error("Invalid snapshot pixel")
      }
      break
    case "sync":
      if (!validVersion(msg.version) || !validCooldown(msg.cooldownMs) || (msg.events !== undefined && (!Array.isArray(msg.events) || msg.events.length > 100000 || !msg.events.every((e:PixelEvent) => validPixel(e) && validVersion(e.version))))) throw new Error("Invalid delta")
      break
    case "pixel_update":
      if (!validPixel(msg) || !validVersion(msg.version)) throw new Error("Invalid pixel")
      break
    case "cooldown":
      if (!validCooldown(msg.cooldownMs)) throw new Error("Invalid cooldown")
      break
    case "accepted":
    case "rejected":
      if (typeof msg.requestId !== "string" || !validCooldown(msg.cooldownMs) || (msg.type === "accepted" && !validVersion(msg.version))) throw new Error("Invalid confirmation")
      break
    case "error":
      if (typeof msg.code !== "string" || !validCooldown(msg.cooldownMs)) throw new Error("Invalid error")
      break
    default: throw new Error("Unknown server message")
  }
  return msg as ServerMessage
}

export function createWS(clientId: string) {
  let ws: WebSocket | null = null
  let version = 0
  let reconnectDelay = 250
  let reconnectTimer: number | null = null
  let handshakeTimer: number | null = null
  let syncTimer: number | null = null
  let syncSendTimer: number | null = null
  let pendingTimer: number | null = null
  let lastSyncSent = -Infinity
  let lastMessage = performance.now()

  function clearSync() {
    if (syncTimer !== null) window.clearTimeout(syncTimer)
    if (syncSendTimer !== null) window.clearTimeout(syncSendTimer)
    syncTimer = syncSendTimer = null
  }
  function finishPending(message: string, accepted = false) {
    if (pendingTimer !== null) window.clearTimeout(pendingTimer)
    pendingTimer = null
    const saved = state.pending
    state.pending = null
    if (accepted && saved?.x === state.selectedPixel?.x && saved?.y === state.selectedPixel?.y) state.selectedPixel = null
    state.message = message
    requestRender()
  }
  function requestSync(recovery = false) {
    if (ws?.readyState !== WebSocket.OPEN) return
    if (recovery) setStatus("syncing")
    if (syncTimer !== null || syncSendTimer !== null) return
    const socket = ws
    syncSendTimer = window.setTimeout(() => {
      syncSendTimer = null
      if (ws !== socket || socket.readyState !== WebSocket.OPEN) return
      lastSyncSent = performance.now()
      socket.send(JSON.stringify({type:"sync",lastVersion:version}))
      syncTimer = window.setTimeout(() => socket.close(),10000)
    }, Math.max(0,1050-(performance.now()-lastSyncSent)))
  }
  function connect() {
    if (!navigator.onLine) { setStatus("disconnected"); return }
    if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) return
    if (reconnectTimer !== null) window.clearTimeout(reconnectTimer)
    reconnectTimer = null
    const url = new URL("/ws",location.href)
    url.protocol = location.protocol === "https:" ? "wss:" : "ws:"
    const socket = new WebSocket(url)
    ws = socket
    let initialized = false
    lastSyncSent = -Infinity
    setStatus("connecting")
    handshakeTimer = window.setTimeout(() => socket.close(),10000)
    socket.onopen = () => {
      if (ws !== socket) return
      setStatus("handshaking")
      lastMessage = performance.now()
      socket.send(JSON.stringify({type:"init_client",clientId,lastVersion:version}))
    }
    socket.onmessage = event => {
      if (ws !== socket) return
      let msg: ServerMessage
      try { msg=parseMessage(event.data) } catch (error) {
        state.message = error instanceof Error ? error.message : "Invalid server response"
        socket.close()
        return
      }
      lastMessage = performance.now()
      switch (msg.type) {
        case "init":
          pixels.clear()
          for (const [key,color] of Object.entries(msg.pixels ?? {})) pixels.set(key,color)
          version=msg.version
          initialized=true
          if (handshakeTimer !== null) window.clearTimeout(handshakeTimer)
          handshakeTimer=null
          clearSync()
          setCooldown(msg.cooldownMs)
          setStatus("connected")
          reconnectDelay=250
          requestRender()
          break
        case "sync":
          if (!initialized) {socket.close(); return}
          clearSync()
          try {
            const result=applyEvents(pixels,version,msg.events ?? [],msg.version)
            version=result.version
            if (result.changed) requestRender()
          } catch { socket.close(); return }
          setCooldown(msg.cooldownMs)
          setStatus("connected")
          break
        case "pixel_update":
          if (!initialized) {socket.close();return}
          if (msg.version <= version) return
          if (msg.version !== version+1) {requestSync(true);return}
          pixels.set(msg.x+":"+msg.y,msg.color)
          version=msg.version
          requestRender()
          break
        case "accepted":
          if (state.pending?.requestId===msg.requestId) finishPending("Pixel saved.",true)
          setCooldown(msg.cooldownMs)
          break
        case "rejected":
          if (state.pending?.requestId===msg.requestId) {
            finishPending(msg.code==="unavailable" ? "Storage unavailable. Your selection is kept." : msg.code==="invalid_pixel" ? "This pixel is invalid." : "Please wait before trying again. Your selection is kept.")
            if (msg.cooldownMs>0) setCooldown(msg.cooldownMs)
          }
          break
        case "cooldown": setCooldown(msg.cooldownMs);break
        case "error":
          clearSync()
          state.message="Sync is busy. Retrying shortly."
          requestFrame()
          syncSendTimer=window.setTimeout(()=>{syncSendTimer=null;requestSync(connection.status==="syncing")},Math.max(1100,msg.cooldownMs))
          break
      }
    }
    socket.onclose=()=>{
      if(ws!==socket)return
      if(handshakeTimer!==null)window.clearTimeout(handshakeTimer)
      handshakeTimer=null
      clearSync()
      ws=null
      if(state.pending)finishPending("Connection lost. Your selection is kept; check the canvas before retrying.")
      setStatus(navigator.onLine ? "reconnecting" : "disconnected")
      if(!navigator.onLine)return
      reconnectTimer=window.setTimeout(connect,reconnectDelay+Math.random()*250)
      reconnectDelay=Math.min(reconnectDelay*2,15000)
    }
    socket.onerror=()=>socket.close()
  }
  function resume(){
    if(!navigator.onLine)return
    if(ws?.readyState===WebSocket.OPEN && performance.now()-lastMessage>45000)ws.close()
    if(!ws || ws.readyState===WebSocket.CLOSED)connect()
  }
  window.addEventListener("online",resume)
  window.addEventListener("offline",()=>{setStatus("disconnected");ws?.close()})
  window.addEventListener("pageshow",resume)
  document.addEventListener("visibilitychange",()=>{if(!document.hidden)resume()})
  window.setInterval(()=>{
    if(ws?.readyState!==WebSocket.OPEN || connection.status!=="connected")return
    if(performance.now()-lastMessage>45000){ws.close();return}
    requestSync()
  },20000)
  connect()
  return {
    placePixel(x:number,y:number,color:string){
      if(ws?.readyState!==WebSocket.OPEN || connection.status!=="connected" || state.pending)return false
      const requestId=crypto.randomUUID()
      state.pending={requestId,x,y,color}
      state.message="Saving pixel…"
      ws.send(JSON.stringify({type:"set_pixel",x,y,color,requestId}))
      pendingTimer=window.setTimeout(()=>{
        finishPending("No confirmation received. Your selection is kept; check the canvas before retrying.")
        ws?.close()
      },10000)
      return true
    }
  }
}
