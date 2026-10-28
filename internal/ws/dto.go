package ws

import "time"

type incomingMessage struct {
	ChatID  int    `json:"chat_id" example:"13"`
	Message string `json:"message" example:"Привет!"`
} //@name WSIncomingMessage

type outgoingMessage struct {
	ID        int       `json:"id" example:"120"`
	ChatID    int       `json:"chat_id" example:"13"`
	UserID    int       `json:"user_id" example:"38"`
	Message   string    `json:"message" example:"Привет!"`
	CreatedAt time.Time `json:"created_at" example:"2026-10-28T18:34:54Z"`
} //@name WSOutgoingMessage
