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

func (s *Service) ListUserChats(
	ctx context.Context,
	userID int,
) ([]ChatModel, error) {
	chats, err := s.repository.ListUserChats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf(
			"list user chats: %w",
			err,
		)
	}

	return chats, nil
}

func (s *Service) AddMember(
	ctx context.Context,
	requesterID, chatID, newUserID int,
) error {
	role, err := s.repository.GetMemberRole(ctx, chatID, requesterID)
	if err != nil {
		if errors.Is(err, ErrNotMember) {
			return fmt.Errorf(
				"chat not found: %w",
				ErrChatNotFound,
			)
		}
		return fmt.Errorf(
			"get member role: %w",
			err,
		)
	}

	if role != "admin" {
		return fmt.Errorf(
			"role invalid: %w",
			ErrForbidden,
		)
	}

	if err := s.repository.AddMember(
		ctx,
		chatID,
		newUserID,
	); err != nil {
		return fmt.Errorf(
			"add member: %w",
			err,
		)
	}

	return nil
}

func (s *Service) RemoveMember(
	ctx context.Context,
	requesterID, chatID, targetID int,
) error {
	if targetID == requesterID {
		return fmt.Errorf(
			"cannot remove yourself: %w",
			ErrInvalidInput,
		)
	}
	role, err := s.repository.GetMemberRole(ctx, chatID, requesterID)
	if err != nil {
		if errors.Is(err, ErrNotMember) {
			return fmt.Errorf(
				"chat not found: %w",
				ErrChatNotFound,
			)
		}
		return fmt.Errorf(
			"get member role: %w",
			err,
		)
	}

	if role != "admin" {
		return fmt.Errorf(
			"role invalid: %w",
			ErrForbidden,
		)
	}

	if err := s.repository.RemoveMember(ctx, chatID, targetID); err != nil {
		return fmt.Errorf(
			"remove member: %w",
			err,
		)
	}
	return nil
}
