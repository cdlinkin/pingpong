package chats

import "time"

type createGroupRequest struct {
	Title string `json:"title"`
}

type createChatResponse struct {
	ID int `json:"id"`
}

type createDirectRequest struct {
	UserID int `json:"user_id"`
}

type chatResponse struct {
	ID        int       `json:"id"`
	Title     *string   `json:"title"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}
