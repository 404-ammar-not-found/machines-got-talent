package game

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

// Client represents one WebSocket connection inside a Hub.
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan WSMessage
	PlayerID  string
	Username  string
	LobbyCode string
}

// Hub manages all WebSocket clients for a single lobby.
type Hub struct {
	mu         sync.RWMutex
	clients    map[string]*Client // keyed by player ID
	broadcast  chan WSMessage
	register   chan *Client
	unregister chan *Client
	onMessage  func(c *Client, msg WSMessage) // injected by Manager
	onLeave    func(playerID string)           // injected by Manager
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		broadcast:  make(chan WSMessage, 256),
		register:   make(chan *Client, 16),
		unregister: make(chan *Client, 16),
	}
}

// Run is the hub event loop — run in its own goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.PlayerID] = client
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.PlayerID]; ok {
				delete(h.clients, client.PlayerID)
				close(client.send)
			}
			h.mu.Unlock()
			if h.onLeave != nil {
				h.onLeave(client.PlayerID)
			}

		case msg := <-h.broadcast:
			h.mu.RLock()
			for _, c := range h.clients {
				select {
				case c.send <- msg:
				default:
					// Slow client — drop the message rather than blocking.
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to every connected client.
func (h *Hub) Broadcast(msg WSMessage) {
	h.broadcast <- msg
}

// Send delivers a message to a specific player (no-op if not connected).
func (h *Hub) Send(playerID string, msg WSMessage) {
	h.mu.RLock()
	c, ok := h.clients[playerID]
	h.mu.RUnlock()
	if !ok {
		return
	}
	select {
	case c.send <- msg:
	default:
	}
}

// ConnectedIDs returns the set of currently connected player IDs.
func (h *Hub) ConnectedIDs() map[string]bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make(map[string]bool, len(h.clients))
	for id := range h.clients {
		out[id] = true
	}
	return out
}

// --- Client read / write pumps ---

// ReadPump reads messages from the WebSocket connection and forwards them to
// the hub's onMessage handler. Must run in its own goroutine.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		var msg WSMessage
		if err := c.conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("ws read error player=%s: %v", c.PlayerID, err)
			}
			break
		}
		if c.hub.onMessage != nil {
			c.hub.onMessage(c, msg)
		}
	}
}

// WritePump drains the client's send channel and writes to the connection.
// Must run in its own goroutine.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteJSON(msg); err != nil {
				log.Printf("ws write error player=%s: %v", c.PlayerID, err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
