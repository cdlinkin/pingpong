package messages

import (
	"context"
	"fmt"
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
}

type Service struct {
	repository repository
}

func NewService(repository repository) *Service {
	return &Service{
		repository: repository,
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
