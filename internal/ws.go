package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

type ClientMsg struct {
	Type        string `json:"type"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	Color       string `json:"color"`
	LastVersion int    `json:"lastVersion"`
	ClientID    string `json:"clientId"`
}

type ServerResponse struct {
	Type        string            `json:"type"`
	Pixels      map[string]string `json:"pixels,omitempty"`
	Events      []Event           `json:"events,omitempty"`
	Version     int               `json:"version"`
	CooldownEnd int64             `json:"cooldownEnd"`
}

func HandleWS(h *Hub, w http.ResponseWriter, r *http.Request) {
	// coder/websocket always accepts the request's own origin. Local Vite uses
	// the same-origin dev proxy, so no cross-origin exceptions are required.
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	conn.SetReadLimit(MaxMessageSize)
	client := h.Add(conn)
	defer func() {
		h.RemoveClient(conn)
		conn.Close(websocket.StatusNormalClosure, "")
	}()
	timer := time.AfterFunc(10*time.Second, func() { h.RemoveClient(conn) })
	defer timer.Stop()
	for {
		_, data, err := conn.Read(client.ctx)
		if err != nil {
			return
		}
		var msg ClientMsg
		if json.Unmarshal(data, &msg) != nil {
			return
		}
		if msg.Type == "init_client" {
			if !ValidClientID(msg.ClientID) || !ValidVersion(msg.LastVersion) {
				return
			}
			if h.GetClientID(conn) != "" {
				continue
			}
			h.SetClientID(conn, msg.ClientID)
			h.stateMu.Lock()
			if h.stopping {
				h.stateMu.Unlock()
				return
			}
			// A newly loaded page has no pixel map, even if localStorage contains an
			// old version. Always start it with a complete authoritative snapshot.
			snap, version := h.Store.Snapshot()
			out, _ := json.Marshal(ServerResponse{Type: "init", Pixels: snap, Version: version, CooldownEnd: h.GetCooldownEnd(msg.ClientID)})
			trySend(client.send, out)
			h.mu.Lock()
			client.ready = true
			h.mu.Unlock()
			h.stateMu.Unlock()
			timer.Stop()
			continue
		}
		id := h.GetClientID(conn)
		if id == "" {
			return
		}
		switch msg.Type {
		case "sync":
			if !ValidVersion(msg.LastVersion) {
				return
			}
			h.stateMu.Lock()
			events, ok := h.Store.GetSince(msg.LastVersion)
			resp := ServerResponse{Type: "sync", Events: events, Version: h.Store.Version(), CooldownEnd: h.GetCooldownEnd(id)}
			if !ok {
				resp.Type = "init"
				resp.Pixels, resp.Version = h.Store.Snapshot()
			}
			out, _ := json.Marshal(resp)
			trySend(client.send, out)
			h.stateMu.Unlock()
		case "set_pixel":
			handleSetPixel(h, id, msg)
		}
	}
}

func handleSetPixel(h *Hub, id string, msg ClientMsg) {
	if !ValidCoords(msg.X, msg.Y) || !ValidColor(msg.Color) {
		return
	}
	h.stateMu.Lock()
	defer h.stateMu.Unlock()
	if h.stopping {
		return
	}
	now := time.Now().UnixMilli()
	if h.GetCooldownEnd(id) > now {
		return
	}
	ev := Event{Version: h.Store.Version() + 1, X: msg.X, Y: msg.Y, Color: msg.Color}
	if err := h.Persistence.Append(JournalEntry{Event: ev, ClientID: id, Time: now}); err != nil {
		h.stopping = true
		log.Printf("pixel persistence failed; writes disabled until restart: %v", err)
		return
	}
	h.Store.Set(fmt.Sprintf("%d:%d", msg.X, msg.Y), msg.X, msg.Y, msg.Color)
	h.UpdateCooldown(id, now, CooldownMs)
	h.lastWrite = time.Now()
	out, _ := json.Marshal(struct {
		Event
		Type string `json:"type"`
	}{Event: ev, Type: "pixel_update"})
	h.Broadcast(out)
	cd, _ := json.Marshal(struct {
		Type string `json:"type"`
		End  int64  `json:"end"`
	}{Type: "cooldown", End: now + CooldownMs})
	h.SendToClientID(id, cd)
}
