import { connectionOverlay, connectionWarning } from "../dom"
import { connection } from "../state"

export function updateConnectionUI(): void {
  if (!connectionOverlay || !connectionWarning) return

  switch (connection.status) {
    case "connected":
      connectionOverlay.style.display = "none"
      return

    case "connecting":
      connectionOverlay.style.display = "flex"
      connectionWarning.textContent =
        "🔄 Connecting socket..."
      return

    case "handshaking":
    case "syncing":
      connectionOverlay.style.display = "flex"
      connectionWarning.textContent =
        "🔄 Syncing canvas..."
      return

    case "reconnecting":
      connectionOverlay.style.display = "flex"
      connectionWarning.textContent =
        "⚠️ Reconnecting..."
      return

    case "disconnected":
      connectionOverlay.style.display = "flex"
      connectionWarning.textContent =
        "❌ Disconnected"
      return
  }
}
