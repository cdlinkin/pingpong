package messages

import (
	"context"
	"fmt"
	"io"
	"os"
)

type repository interface {
	IsMember(ctx context.Context, chatID, userID int) (bool, error)
	CreateMessage(ctx context.Context, chatID, userID int, message string, replyToID *int) (MessageModel, error)
	ListMessage(ctx context.Context, chatID, beforeID, limit int) ([]MessageModel, error)
	SearchMessages(ctx context.Context, chatID int, query string) ([]MessageModel, error)
	ListMemberIDs(ctx context.Context, chatID int) ([]int, error)
	UpdateMessage(ctx context.Context, chatID, msgID, userID int, msg string) (MessageModel, error)
	DeleteMessage(ctx context.Context, chatID, msgID, userID int) error
	MarkRead(ctx context.Context, chatID, userID, msgID int) (int, error)
	ListReads(ctx context.Context, chatID int) ([]ReadModel, error)
	MessageInChat(ctx context.Context, chatID, msgID int) (bool, error)
	RemoveReaction(ctx context.Context, msgID, userID int, emoji string) (bool, error)
	UpsertReaction(ctx context.Context, msgID, userID int, emoji string) error
	ListReactions(ctx context.Context, msgIDs []int) ([]ReactionModel, error)
	CreateAttachmentMessage(ctx context.Context, chatID, userID int, caption string, replyToID *int, a AttachmentModel) (MessageModel, error)
	ListAttachments(ctx context.Context, msgIDs []int) ([]AttachmentModel, error)
	GetAttachment(ctx context.Context, id int) (AttachmentModel, error)
}

type fileStorage interface {
	Save(r io.Reader) (string, int64, error)
	Open(key string) (*os.File, error)
	Remove(key string) error
}

type Service struct {
	repository repository
	storage    fileStorage
}

func NewService(repository repository, storage fileStorage) *Service {
	return &Service{
		repository: repository,
		storage:    storage,
	}
}

func (s *Service) ListMemberIDs(
	ctx context.Context,
	chatID int,
) ([]int, error) {
	usersID, err := s.repository.ListMemberIDs(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf(
			"list member ids: %w",
			err,
		)
	}

	return usersID, nil
}

func (s *Service) ensureMember(ctx context.Context, chatID, userID int) error {
	isMember, err := s.repository.IsMember(ctx, chatID, userID)
	if err != nil {
		return fmt.Errorf(
			"is member: %w",
			err,
		)
	}
	if !isMember {
		return fmt.Errorf(
			"user not in chat: %w",
			ErrChatNotFound,
		)
	}
	return nil
}

func (s *Service) ensureReplyTarget(ctx context.Context, chatID int, replyToID *int) error {
	if replyToID == nil {
		return nil
	}
	ok, err := s.repository.MessageInChat(ctx, chatID, *replyToID)
	if err != nil {
		return fmt.Errorf(
			"message in chat: %w",
			err,
		)
	}
	if !ok {
		return fmt.Errorf(
			"reply target not in chat: %w",
			ErrMessageNotFound,
		)
	}
	return nil
}
