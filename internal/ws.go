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
	RequestID   string `json:"requestId,omitempty"`
}

type ServerResponse struct {
	RequestID   string          `json:"requestId,omitempty"`
	Code        string          `json:"code,omitempty"`
	Protocol    int             `json:"protocol,omitempty"`
	GridSize    int             `json:"gridSize,omitempty"`
	End         int64           `json:"end,omitempty"`
	Type        string          `json:"type"`
	Pixels      json.RawMessage `json:"pixels,omitempty"`
	Events      []Event         `json:"events,omitempty"`
	Version     int             `json:"version"`
	CooldownMs  int64           `json:"cooldownMs"`
	CooldownEnd int64           `json:"cooldownEnd"`
}

func HandleWS(h *Hub, w http.ResponseWriter, r *http.Request) {
	if !h.Ready() {
		http.Error(w, "server unavailable", 503)
		return
	}
	source := sourceKey(r)
	if !h.admit(source, time.Now().UnixMilli()) {
		h.connectionRejects.Add(1)
		w.Header().Set("Retry-After", "1")
		http.Error(w, "connection budget exceeded", 429)
		return
	}
	defer h.releaseAdmission(source)
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
		next, retry := advanceBudget(client.messageBudget, time.Now().UnixMilli(), 100, 20)
		if retry > 0 {
			conn.Close(websocket.StatusPolicyViolation, "message budget exceeded")
			return
		}
		client.messageBudget = next
		msg, err := decodeClientMsg(data)
		if err != nil {
			conn.Close(websocket.StatusPolicyViolation, "invalid message")
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
			if h.stopping || !h.Ready() {
				h.stateMu.Unlock()
				return
			}
			// A newly loaded page has no pixel map, even if localStorage contains an
			// old version. Always start it with a complete authoritative snapshot.
			snap, version := h.snapshotLocked()
			end := h.GetCooldownEnd(msg.ClientID)
			out, _ := json.Marshal(ServerResponse{Protocol: Protocol.Protocol, GridSize: GridSize, Type: "init", Pixels: snap, Version: version, CooldownEnd: end, CooldownMs: max(0, end-time.Now().UnixMilli())})
			if !h.enqueue(client, out) {
				h.stateMu.Unlock()
				return
			}
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
			h.syncs.Add(1)
			if !ValidVersion(msg.LastVersion) {
				return
			}
			now := time.Now().UnixMilli()
			if now-client.lastSync < 1000 {
				h.sendRateLimit(client)
				continue
			}
			client.lastSync = now
			h.stateMu.Lock()
			events, ok := h.store.GetSince(msg.LastVersion)
			end := h.GetCooldownEnd(id)
			resp := ServerResponse{Type: "sync", Events: events, Version: h.store.Version(), CooldownEnd: end, CooldownMs: max(0, end-time.Now().UnixMilli())}
			if !ok {
				next, retry := advanceBudget(h.snapshotBudget, now, 500, 2)
				if retry > 0 {
					h.stateMu.Unlock()
					h.sendRateLimit(client)
					continue
				}
				h.snapshotBudget = next
				resp.Type = "init"
				resp.Protocol = Protocol.Protocol
				resp.GridSize = GridSize
				resp.Pixels, resp.Version = h.snapshotLocked()
			}
			out, _ := json.Marshal(resp)
			h.enqueue(client, out)
			h.stateMu.Unlock()
		case "set_pixel":
			response := handleSetPixel(h, id, source, msg)
			if msg.RequestID != "" {
				response.RequestID = msg.RequestID
				out, _ := json.Marshal(response)
				h.enqueue(client, out)
			}
		}
	}
}

// Called under stateMu so init ordering remains atomic with live broadcasts.
func (h *Hub) snapshotLocked() (json.RawMessage, int) {
	version := h.store.Version()
	if h.snapshotVersion != version {
		pixels, _ := h.store.Snapshot()
		h.snapshotJSON = encodePixels(pixels)
		h.snapshotVersion = version
	}
	return h.snapshotJSON, version
}

func (h *Hub) sendRateLimit(client *Client) {
	out, _ := json.Marshal(ServerResponse{Type: "error", Code: "rate_limited", CooldownMs: 1000})
	h.enqueue(client, out)
}

func decodeClientMsg(data []byte) (ClientMsg, error) {
	var msg ClientMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return msg, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return msg, err
	}
	required := []string{"type"}
	switch msg.Type {
	case "set_pixel":
		required = append(required, "x", "y", "color")
	case "init_client":
		required = append(required, "clientId", "lastVersion")
	case "sync":
		required = append(required, "lastVersion")
	default:
		return msg, fmt.Errorf("unknown message type")
	}
	for _, key := range required {
		if value, ok := fields[key]; !ok || string(value) == "null" {
			return msg, fmt.Errorf("missing %s", key)
		}
	}
	if msg.RequestID != "" && !ValidClientID(msg.RequestID) {
		return msg, fmt.Errorf("invalid request id")
	}
	return msg, nil
}

func handleSetPixel(h *Hub, id, source string, msg ClientMsg) (response ServerResponse) {
	defer func() {
		if response.Type == "accepted" {
			h.accepted.Add(1)
		} else {
			h.rejected.Add(1)
		}
	}()
	if !ValidCoords(msg.X, msg.Y) || !ValidColor(msg.Color) {
		return ServerResponse{Type: "rejected", Code: "invalid_pixel"}
	}
	h.stateMu.Lock()
	defer h.stateMu.Unlock()
	if h.stopping || !h.Ready() {
		return ServerResponse{Type: "rejected", Code: "unavailable"}
	}
	now := time.Now().UnixMilli()
	if h.GetCooldownEnd(id) > now {
		h.sendCooldown(id)
		return ServerResponse{Type: "rejected", Code: "cooldown", CooldownMs: max(0, h.GetCooldownEnd(id)-now)}
	}
	sourceNext, retry := h.sourceCooldown(source, now)
	globalNext, globalRetry := advanceBudget(h.globalWrite, now, 100, 20)
	if source == "" {
		globalRetry = 0
	}
	if retry > 0 || globalRetry > 0 {
		return ServerResponse{Type: "rejected", Code: "rate_limited", CooldownMs: max(retry, globalRetry)}
	}
	ev := Event{Version: h.store.Version() + 1, X: msg.X, Y: msg.Y, Color: msg.Color}
	watchdog := time.AfterFunc(5*time.Second, func() { h.markFailed(fmt.Errorf("journal fsync exceeded 5 seconds")) })
	defer watchdog.Stop()
	if err := h.persistence.Append(JournalEntry{Event: ev, ClientID: id, Time: now, Source: source, SourceTime: sourceNext}); err != nil {
		h.stopping = true
		h.markFailed(err)
		log.Printf("pixel persistence failed; writes disabled until restart: %v", err)
		return ServerResponse{Type: "rejected", Code: "unavailable"}
	}
	if !h.Ready() {
		return ServerResponse{Type: "rejected", Code: "unavailable"}
	}
	if source != "" {
		h.sourceActions[source] = sourceNext
	}
	h.globalWrite = globalNext
	h.store.Set(fmt.Sprintf("%d:%d", msg.X, msg.Y), msg.X, msg.Y, msg.Color)
	h.UpdateCooldown(id, now, CooldownMs)
	h.lastWrite = time.Now()
	out, _ := json.Marshal(struct {
		Event
		Type string `json:"type"`
	}{Event: ev, Type: "pixel_update"})
	h.Broadcast(out)
	h.sendCooldown(id)
	return ServerResponse{Type: "accepted", Version: ev.Version, CooldownMs: CooldownMs}
}

func (h *Hub) sendCooldown(id string) {
	end := h.GetCooldownEnd(id)
	cd, _ := json.Marshal(ServerResponse{Type: "cooldown", End: end, CooldownMs: max(0, end-time.Now().UnixMilli())})
	h.SendToClientID(id, cd)
}
