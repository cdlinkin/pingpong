package chats

import (
	"context"
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
