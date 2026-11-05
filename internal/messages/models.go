package messages

import "time"

type MessageModel struct {
	ID            int
	ChatID        int
	UserID        int
	Message       string
	CreatedAt     time.Time
	EditedAt      *time.Time
	DeletedAt     *time.Time
	ReplyToID     *int
	ReplyUserID   *int
	ReplyMessage  *string
	ReplyFileName *string
	Reactions     []ReactionModel
	Attachment    *AttachmentModel
}

type AttachmentModel struct {
	ID          int
	MsgID       int
	ChatID      int
	FileName    string
	ContentType string
	Size        int64
	StorageKey  string
	CreatedAt   time.Time
	Deleted     bool
}

type ReadModel struct {
	UserID     int
	LastReadID int
}

type ReactionModel struct {
	MsgID  int
	UserID int
	Emoji  string
}
