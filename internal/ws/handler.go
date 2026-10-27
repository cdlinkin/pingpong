package ws

import (
	"log"
	"net/http"

	"github.com/cdlinkin/pingpong/internal/auth"
	"github.com/gorilla/websocket"
)

type Handler struct {
	secret []byte
	hub    *Hub
}

func NewHandler(secret []byte, hub *Hub) *Handler {
	return &Handler{
		secret: secret,
		hub:    hub,
	}
}

func (h *Handler) ServeWS(rw http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	id, err := auth.ParseToken(token, h.secret)
	if err != nil {
		http.Error(
			rw,
			"invalid token",
			http.StatusUnauthorized,
		)
		return
	}
	upgrader := websocket.Upgrader{}

	conn, err := upgrader.Upgrade(rw, r, nil)
	if err != nil {
		log.Printf("upgrader conn")
		return
	}
	log.Printf("ws: user %d connected", id)
	defer conn.Close()

	client := &Client{
		Conn:   conn,
		UserID: id,
		Send:   make(chan []byte, 16),
	}

	h.hub.Register(client)
	defer h.hub.Unregister(client)

	go writePump(client)

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("ws: user %d disconnected: %v", id, err)
			return
		}

		h.hub.SendToUser(id, data)
	}
}

func writePump(c *Client) {
	for data := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("client write message: %v", err)
			return
		}
	}
}
