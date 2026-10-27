package ws

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn   *websocket.Conn
	UserID int
	Send   chan []byte
}

type Hub struct {
	mu   sync.Mutex
	pool map[int]map[*Client]bool
}

func NewHub() *Hub {
	return &Hub{
		pool: make(map[int]map[*Client]bool),
	}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.pool[c.UserID] == nil {
		h.pool[c.UserID] = make(map[*Client]bool)
	}
	h.pool[c.UserID][c] = true
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.pool[c.UserID][c]; !ok {
		return
	}
	delete(h.pool[c.UserID], c)
	if len(h.pool[c.UserID]) == 0 {
		delete(h.pool, c.UserID)
	}
	close(c.Send)
}

func (h *Hub) SendToUser(userID int, data []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.pool[userID] {
		select {
		case c.Send <- data:
		default:
		}
	}
}
