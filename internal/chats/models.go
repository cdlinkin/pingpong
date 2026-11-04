package chats

import "time"

type ChatModel struct {
	ID          int
	Title       *string
	Type        string
	CreatedAt   time.Time
	UnreadCount int
	PeerLogin   *string
}

type MemberModel struct {
	UserID int
	Login  string
	Role   string
}
