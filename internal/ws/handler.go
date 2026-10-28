package ws

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/cdlinkin/pingpong/internal/auth"
	"github.com/cdlinkin/pingpong/internal/messages"
	"github.com/gorilla/websocket"
)

type Handler struct {
	secret     []byte
	hub        *Hub
	msgService *messages.Service
}

func NewHandler(
	secret []byte,
	hub *Hub,
	msgService *messages.Service,
) *Handler {
	return &Handler{
		secret:     secret,
		hub:        hub,
		msgService: msgService,
	}
}

// ServeWS godoc
//
//	@Summary		WebSocket: сообщения в реальном времени
//	@Description	Открывает WebSocket. Swagger UI не умеет подключаться к WebSocket: проверяйте в Postman (New → WebSocket) или через фронтенд.
//	@Description
//	@Description	**Подключение:** `ws://localhost:8080/ws?token=<JWT>`. Токен передаётся в query, потому что браузер не может ставить заголовки при открытии WebSocket.
//	@Description
//	@Description	**Клиент → сервер:** `{"chat_id": 13, "message": "Привет!"}` (модель WSIncomingMessage).
//	@Description
//	@Description	**Сервер → клиент:** сохранённое сообщение (модель WSOutgoingMessage) рассылается всем участникам чата, которые сейчас онлайн, включая отправителя. Для него это подтверждение доставки.
//	@Description
//	@Description	**Ошибки** приходят только отправителю: `{"error": "bad json"}`, `{"error": "invalid message"}`, `{"error": "chat not found"}`, `{"error": "internal error"}`.
//	@Tags			realtime
//	@Param			token	query		string			true	"JWT из /login"
//	@Param			message	body		incomingMessage	false	"Формат сообщения клиента (отправляется по WebSocket, не в теле HTTP)"
//	@Success		101		{object}	outgoingMessage	"Соединение переключено на WebSocket; так выглядят входящие сообщения"
//	@Failure		401		{string}	string			"invalid token"
//	@Router			/ws [get]
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
		var in incomingMessage
		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("ws: user %d disconnected: %v", id, err)
			return
		}

		if err := json.Unmarshal(data, &in); err != nil {
			trySend(client, []byte(`{"error":"bad json"}`))
			continue
		}
		msgDomain, err := h.msgService.SendMessage(
			r.Context(),
			in.ChatID,
			id,
			in.Message,
		)
		if err != nil {
			switch {
			case errors.Is(err, messages.ErrInvalidInput):
				trySend(client, []byte(`{"error":"invalid message"}`))
			case errors.Is(err, messages.ErrChatNotFound):
				trySend(client, []byte(`{"error":"chat not found"}`))
			default:
				log.Printf("ws: send message: %v", err)
				trySend(client, []byte(`{"error":"internal error"}`))
			}
			continue
		}
		response := outgoingMessage{
			ID:        msgDomain.ID,
			ChatID:    msgDomain.ChatID,
			UserID:    msgDomain.UserID,
			Message:   msgDomain.Message,
			CreatedAt: msgDomain.CreatedAt,
		}
		data, err = json.Marshal(response)
		if err != nil {
			log.Printf("ws: marshal response: %v", err)
			trySend(client, []byte(`{"error":"internal error"}`))
			continue
		}
		ids, err := h.msgService.ListMemberIDs(r.Context(), msgDomain.ChatID)
		if err != nil {
			log.Printf("ws: list member ids: %v", err)
			trySend(client, []byte(`{"error":"internal error"}`))
			continue
		}
		for _, memberID := range ids {
			h.hub.SendToUser(memberID, data)
		}
	}
}

func trySend(c *Client, data []byte) {
	select {
	case c.Send <- data:
	default:
		log.Printf("ws: user %d send buffer full, drop message", c.UserID)
	}
}

func writePump(c *Client) {
	for data := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("client write message: %v", err)
			c.Conn.Close()
			return
		}
	}
}
