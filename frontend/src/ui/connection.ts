import { connectionOverlay, connectionWarning } from "../dom"
import { connection } from "../state"
import type { ConnectionStatus } from "../state"
let previous: ConnectionStatus | null = null
export function updateConnectionUI(): void {
  if (previous === connection.status) return
  previous = connection.status
  const messages: Record<ConnectionStatus, string> = {
    connected: "Connected",
    connecting: "Connecting…",
    handshaking: "Syncing canvas…",
    syncing: "Recovering canvas…",
    reconnecting: "Reconnecting…",
    disconnected: "Offline. Reconnect to place pixels."
  }
  connectionOverlay.style.display =
    connection.status === "connected" ? "none" : "flex"
  connectionWarning.textContent = messages[connection.status]
  const status = document.getElementById("connection-status")
  if (status) status.textContent = messages[connection.status]
}
