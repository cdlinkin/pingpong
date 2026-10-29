package messages

import "time"

type MessageModel struct {
	ID        int
	ChatID    int
	UserID    int
	Message   string
	CreatedAt time.Time
	EditedAt  *time.Time
	DeletedAt *time.Time
}
