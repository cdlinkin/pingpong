package chats

import (
	"context"
	"errors"
	"fmt"
)

type Service struct {
	repository *Repository
}

func NewService(
	repository *Repository,
) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) CreateGroupChat(
	ctx context.Context,
	title string,
	creatorID int,
) (int, error) {
	title, err := validateTitle(title)
	if err != nil {
		return 0, fmt.Errorf(
			"validate title: %w",
			err,
		)
	}

	chatID, err := s.repository.CreateGroupChat(ctx, title, creatorID)
	if err != nil {
		return 0, fmt.Errorf(
			"create group chat: %w",
			err,
		)
	}

	return chatID, nil
}

func (s *Service) CreateDirectChat(
	ctx context.Context,
	creatorID, otherID int,
) (int, bool, error) {
	if creatorID == otherID {
		return 0, false, fmt.Errorf(
			"can’t create a chat with yourself: %w",
			ErrInvalidInput,
		)
	}

	chatID, err := s.repository.FindDirectChat(ctx, creatorID, otherID)
	if err == nil {
		return chatID, false, nil
	}
	if !errors.Is(err, ErrChatNotFound) {
		return 0, false, fmt.Errorf("find direct chat: %w", err)
	}
	chatID, err = s.repository.CreateDirectChat(ctx, creatorID, otherID)
	if err != nil {
		return 0, false, fmt.Errorf(
			"create direct chat: %w",
			err,
		)
	}
	return chatID, true, nil
}
