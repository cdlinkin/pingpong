package messages

import "time"

type msgRequest struct {
	Message string `json:"message"`
}

type msgResponse struct {
	ID        int       `json:"id"`
	ChatID    int       `json:"chat_id"`
	UserID    int       `json:"user_id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}
