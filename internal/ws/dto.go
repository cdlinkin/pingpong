package ws

import "time"

type incomingMessage struct {
	ChatID  int    `json:"chat_id"`
	Message string `json:"message"`
}

type outgoingMessage struct {
	ID        int       `json:"id"`
	ChatID    int       `json:"chat_id"`
	UserID    int       `json:"user_id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}
