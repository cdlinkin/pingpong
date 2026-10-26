package messages

import (
	"context"
	"fmt"
	"math"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
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
