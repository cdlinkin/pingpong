package messages

import "time"

type msgRequest struct {
	Message string `json:"message" example:"Привет!" minLength:"1" maxLength:"500"`
} //@name MessageRequest

type msgResponse struct {
	ID        int        `json:"id" example:"120"`
	ChatID    int        `json:"chat_id" example:"13"`
	UserID    int        `json:"user_id" example:"38"`
	Message   string     `json:"message" example:"Привет!"`
	CreatedAt time.Time  `json:"created_at" example:"2026-10-28T18:34:54Z"`
	EditedAt  *time.Time `json:"edited_at,omitempty" example:"2026-10-28T18:40:12Z"`
	Deleted   bool       `json:"deleted" example:"false"`
} //@name Message

type editedEvent struct {
	Type    string      `json:"type"`
	Message msgResponse `json:"message"`
}

type deletedEvent struct {
	Type   string `json:"type"`
	ChatID int    `json:"chat_id"`
	MsgID  int    `json:"id"`
}
