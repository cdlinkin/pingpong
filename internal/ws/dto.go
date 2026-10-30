package ws

import "time"

type incomingMessage struct {
	ChatID    int    `json:"chat_id" example:"13"`
	Message   string `json:"message" example:"Привет!"`
	ReplyToID *int   `json:"reply_to_id,omitempty" example:"118"`
} //@name WSIncomingMessage

type outgoingMessage struct {
	ID        int       `json:"id" example:"120"`
	ChatID    int       `json:"chat_id" example:"13"`
	UserID    int       `json:"user_id" example:"38"`
	Message   string    `json:"message" example:"Привет!"`
	CreatedAt time.Time `json:"created_at" example:"2026-10-28T18:34:54Z"`
	ReplyTo   *replyOut `json:"reply_to,omitempty"`
} //@name WSOutgoingMessage

type replyOut struct {
	ID      int    `json:"id" example:"118"`
	UserID  int    `json:"user_id" example:"39"`
	Message string `json:"message" example:"Во сколько встречаемся?"`
} //@name WSReply
