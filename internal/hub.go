package internal

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

type Hub struct {
	saveMu          sync.Mutex
	savedVersion    int
	lastSave        atomic.Int64
	snapshotVersion int
	snapshotJSON    json.RawMessage
	stateMu         sync.Mutex
	lastWrite       time.Time
	sourceActions   map[string]int64 // guarded by stateMu
	globalWrite     int64
	snapshotBudget  int64
	admissions      map[string]*admission // guarded by mu
	openConnections int
	handshakeBudget int64
	stopping        bool
	failed          atomic.Bool
	closing         atomic.Bool
	failures        chan error
	mu              sync.RWMutex
	clients         map[*websocket.Conn]*Client
	store           *PixelStore
	lastAction      map[string]int64
	connToID        map[*websocket.Conn]string
	idToClients     map[string]map[*Client]struct{}
	persistence     *Persistence
}

type Client struct {
	queued        atomic.Int64
	messageBudget int64
	lastSync      int64
	ready         bool
	conn          *websocket.Conn
	send          chan []byte
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewHub() *Hub {
	dir := os.Getenv("STATE_DIRECTORY")
	if dir == "" {
		dir = "./data"
	}
	h, err := OpenHub(dir)
	if err != nil {
		panic(err)
	}
	return h
}

func OpenHub(dir string) (*Hub, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	p := NewPersistence(filepath.Join(dir, "snapshot.json"))
	state, err := p.Load()
	if err != nil {
		return nil, err
	}
	store := NewPixelStore()
	store.LoadSnapshot(state.Pixels, state.Version)
	return &Hub{
		clients: make(map[*websocket.Conn]*Client), store: store,
		lastAction: state.LastAction, connToID: make(map[*websocket.Conn]string),
		idToClients: make(map[string]map[*Client]struct{}), persistence: p,
		lastWrite:       time.Now(),
		sourceActions:   state.SourceActions,
		failures:        make(chan error, 1),
		admissions:      make(map[string]*admission),
		savedVersion:    -1,
		snapshotVersion: -1,
	}, nil
}

func (h *Hub) Add(c *websocket.Conn) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		conn:   c,
		send:   make(chan []byte, 32),
		ctx:    ctx,
		cancel: cancel,
	}

	h.mu.Lock()
	h.clients[c] = client
	h.mu.Unlock()

	go func(cl *Client) {
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		defer func() {
			h.RemoveClient(cl.conn)
			cl.conn.Close(websocket.StatusNormalClosure, "")
		}()

		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(cl.ctx, 10*time.Second)
				err := cl.conn.Ping(ctx)
				cancel()
				if err != nil {
					return
				}
			case msg, ok := <-cl.send:
				if !ok {
					return
				}

				ctx, cancel := context.WithTimeout(cl.ctx, 5*time.Second)
				err := cl.conn.Write(ctx, websocket.MessageText, msg)
				cl.queued.Add(-int64(len(msg)))
				cancel()

				if err != nil {
					return
				}

			case <-cl.ctx.Done():
				return
			}
		}
	}(client)

	return client
}
func (h *Hub) Broadcast(msg []byte) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for _, client := range h.clients {
		if client.ready {
			clients = append(clients, client)
		}
	}
	h.mu.RUnlock()

	for _, client := range clients {
		h.enqueue(client, msg)
	}
}

func (h *Hub) SendToClientID(id string, msg []byte) {
	h.mu.RLock()
	clientSet, ok := h.idToClients[id]
	if !ok {
		h.mu.RUnlock()
		return
	}

	clients := make([]*Client, 0, len(clientSet))
	for client := range clientSet {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		h.enqueue(client, msg)
	}
}

func (h *Hub) GetClientID(conn *websocket.Conn) string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.connToID[conn]
}

func (h *Hub) SetClientID(conn *websocket.Conn, id string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	client := h.clients[conn]
	if client == nil {
		return
	}

	oldID := h.connToID[conn]
	if oldID == id {
		return
	}

	if oldID != "" && oldID != id {
		if oldSet, ok := h.idToClients[oldID]; ok {
			delete(oldSet, client)

			if len(oldSet) == 0 {
				delete(h.idToClients, oldID)
			}
		}
	}

	h.connToID[conn] = id

	if h.idToClients[id] == nil {
		h.idToClients[id] = make(map[*Client]struct{})
	}

	h.idToClients[id][client] = struct{}{}
}

func (h *Hub) UpdateCooldown(id string, now int64, cooldown int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	last := h.lastAction[id]

	if now-last < cooldown {
		return false
	}

	h.lastAction[id] = now
	return true
}

func (h *Hub) GetCooldownEnd(id string) int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	last := h.lastAction[id]

	if last == 0 {
		return 0
	}

	return last + CooldownMs
}

func (h *Hub) RemoveClient(conn *websocket.Conn) {
	h.mu.Lock()

	client := h.clients[conn]
	id := h.connToID[conn]

	if client != nil {
		delete(h.clients, conn)
		delete(h.connToID, conn)

		if id != "" {
			if clients, ok := h.idToClients[id]; ok {
				delete(clients, client)
				if len(clients) == 0 {
					delete(h.idToClients, id)
				}
			}
		}
	}

	h.mu.Unlock()

	if client != nil {
		client.cancel()
	}
}

func (h *Hub) enqueue(client *Client, msg []byte) bool {
	if client.queued.Add(int64(len(msg))) > maxQueuedBytes {
		client.queued.Add(-int64(len(msg)))
		h.RemoveClient(client.conn)
		return false
	}
	select {
	case client.send <- msg:
		return true
	default:
		client.queued.Add(-int64(len(msg)))
		h.RemoveClient(client.conn)
		return false
	}
}

func (h *Hub) CleanupLastAction(maxAge int64) {
	now := time.Now().UnixMilli()

	h.mu.Lock()
	defer h.mu.Unlock()

	for id, t := range h.lastAction {
		if now-t > maxAge {
			delete(h.lastAction, id)
		}
	}
}
