package messages

import (
	"context"
	"fmt"
	"math"
)

type repository interface {
	IsMember(ctx context.Context, chatID, userID int) (bool, error)
	CreateMessage(ctx context.Context, chatID, userID int, message string) (MessageModel, error)
	ListMessage(ctx context.Context, chatID, beforeID, limit int) ([]MessageModel, error)
	SearchMessages(ctx context.Context, chatID int, query string) ([]MessageModel, error)
	ListMemberIDs(ctx context.Context, chatID int) ([]int, error)
	UpdateMessage(ctx context.Context, chatID, msgID, userID int, msg string) (MessageModel, error)
	DeleteMessage(ctx context.Context, chatID, msgID, userID int) error
	MarkRead(ctx context.Context, chatID, userID, msgID int) (int, error)
	ListReads(ctx context.Context, chatID int) ([]ReadModel, error)
}

type Service struct {
	repository repository
}

func NewService(repository repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) SendMessage(
	ctx context.Context,
	chatID, userID int,
	msg string,
) (MessageModel, error) {
	msg, err := validateMsg(msg)
	if err != nil {
		return MessageModel{}, fmt.Errorf(
			"validate message: %w",
			err,
		)
	}

	isMember, err := s.repository.IsMember(ctx, chatID, userID)
	if err != nil {
		return MessageModel{}, fmt.Errorf(
			"is member: %w",
			err,
		)
	}
	if !isMember {
		return MessageModel{}, fmt.Errorf(
			"user not in chat: %w",
			ErrChatNotFound,
		)
	}

	msgModel, err := s.repository.CreateMessage(ctx, chatID, userID, msg)
	if err != nil {
		return MessageModel{}, fmt.Errorf(
			"create message: %w",
			err,
		)
	}

	return msgModel, nil
}

func (s *Service) ListMessage(
	ctx context.Context,
	userID, chatID int,
	beforeID, limit int,
) ([]MessageModel, error) {
	isMember, err := s.repository.IsMember(ctx, chatID, userID)
	if err != nil {
		return nil, fmt.Errorf(
			"is member: %w",
			err,
		)
	}
	if !isMember {
		return nil, fmt.Errorf(
			"user not in chat: %w",
			ErrChatNotFound,
		)
	}

	if beforeID == 0 {
		beforeID = math.MaxInt32
	}

	if limit <= 0 {
		limit = 50
	}
	if limit >= 100 {
		limit = 100
	}
	messages, err := s.repository.ListMessage(ctx, chatID, beforeID, limit)
	if err != nil {
		return nil, fmt.Errorf(
			"list messages: %w",
			err,
		)
	}

	return messages, nil
}

func (s *Service) SearchMessages(
	ctx context.Context,
	userID, chatID int,
	query string,
) ([]MessageModel, error) {
	query, err := validateQuery(query)
	if err != nil {
		return nil, fmt.Errorf(
			"validate query: %w",
			err,
		)
	}
	isMember, err := s.repository.IsMember(ctx, chatID, userID)
	if err != nil {
		return nil, fmt.Errorf(
			"is member: %w",
			err,
		)
	}
	if !isMember {
		return nil, fmt.Errorf(
			"user not in chat: %w",
			ErrChatNotFound,
		)
	}

	messages, err := s.repository.SearchMessages(ctx, chatID, query)
	if err != nil {
		return nil, fmt.Errorf(
			"search messages: %w",
			err,
		)
	}

	return messages, nil
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

func (s *Service) EditMessage(
	ctx context.Context,
	userID, chatID, msgID int,
	msg string,
) (MessageModel, error) {
	msg, err := validateMsg(msg)
	if err != nil {
		return MessageModel{}, fmt.Errorf(
			"validate message: %w",
			err,
		)
	}

	isMember, err := s.repository.IsMember(ctx, chatID, userID)
	if err != nil {
		return MessageModel{}, fmt.Errorf(
			"is member: %w",
			err,
		)
	}
	if !isMember {
		return MessageModel{}, fmt.Errorf(
			"user not in chat: %w",
			ErrChatNotFound,
		)
	}

	msgModel, err := s.repository.UpdateMessage(ctx, chatID, msgID, userID, msg)
	if err != nil {
		return MessageModel{}, fmt.Errorf(
			"update message: %w",
			err,
		)
	}

	return msgModel, nil
}

func (s *Service) DeleteMessage(
	ctx context.Context,
	userID, chatID, msgID int,
) error {
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

	if err := s.repository.DeleteMessage(ctx, chatID, msgID, userID); err != nil {
		return fmt.Errorf(
			"delete message: %w",
			err,
		)
	}

	return nil
}

func (s *Service) MarkRead(
	ctx context.Context,
	chatID, userID, msgID int,
) (int, error) {
	if msgID <= 0 {
		return 0, fmt.Errorf(
			"message id invalid input: %w",
			ErrInvalidInput,
		)
	}

	lastReadID, err := s.repository.MarkRead(ctx, chatID, userID, msgID)
	if err != nil {
		return 0, fmt.Errorf(
			"mark read: %w",
			err,
		)
	}

	return lastReadID, nil
}

func (s *Service) ListReads(
	ctx context.Context,
	userID, chatID int,
) ([]ReadModel, error) {
	isMember, err := s.repository.IsMember(ctx, chatID, userID)
	if err != nil {
		return nil, fmt.Errorf(
			"is member: %w",
			err,
		)
	}
	if !isMember {
		return nil, fmt.Errorf(
			"user not in chat: %w",
			ErrChatNotFound,
		)
	}

	reads, err := s.repository.ListReads(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf(
			"list reads: %w",
			err,
		)
	}

	return reads, nil
}
