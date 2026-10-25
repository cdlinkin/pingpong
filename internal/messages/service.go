package messages

import (
	"context"
	"fmt"
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
