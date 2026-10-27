package ws

import (
	"log"
	"net/http"

	"github.com/cdlinkin/pingpong/internal/auth"
	"github.com/gorilla/websocket"
)

type Handler struct {
	secret []byte
}

func NewHandler(secret []byte) *Handler {
	return &Handler{
		secret: secret,
	}
}

func (h *Handler) ServeWS(rw http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	id, err := auth.ParseToken(token, h.secret)
	if err != nil {
		log.Printf("ws: user %d connected", id)
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
	defer conn.Close()

	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("ws: user %d disconnected: %v", id, err)
			return
		}

		if err := conn.WriteMessage(msgType, data); err != nil {
			return
		}
	}
}
