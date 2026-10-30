package messages

import "time"

type MessageModel struct {
	ID           int
	ChatID       int
	UserID       int
	Message      string
	CreatedAt    time.Time
	EditedAt     *time.Time
	DeletedAt    *time.Time
	ReplyToID    *int
	ReplyUserID  *int
	ReplyMessage *string
}

type ReadModel struct {
	UserID     int
	LastReadID int
}
