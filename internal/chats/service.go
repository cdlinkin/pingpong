package chats

import (
	"context"
	"errors"
	"fmt"
)

type repository interface {
	CreateGroupChat(ctx context.Context, title string, creatorID int) (int, error)
	CreateDirectChat(ctx context.Context, userA, userB int) (int, error)
	FindDirectChat(ctx context.Context, userA, userB int) (int, error)
	ListUserChats(ctx context.Context, userID int) ([]ChatModel, error)
	GetMemberRole(ctx context.Context, chatID, userID int) (string, error)
	AddMember(ctx context.Context, chatID, userID int) error
	RemoveMember(ctx context.Context, chatID, userID int) error
	ListMembers(ctx context.Context, chatID int) ([]MemberModel, error)
	SetMuted(ctx context.Context, userID, chatID int, muted bool) error
}

type Service struct {
	repository repository
}

func NewService(
	repository repository,
) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) memberRole(ctx context.Context, chatID, userID int) (string, error) {
	role, err := s.repository.GetMemberRole(ctx, chatID, userID)
	if err != nil {
		if errors.Is(err, ErrNotMember) {
			return "", fmt.Errorf(
				"chat not found: %w",
				ErrChatNotFound,
			)
		}
		return "", fmt.Errorf(
			"get member role: %w",
			err,
		)
	}
	return role, nil
}

func (s *Service) ensureAdmin(ctx context.Context, chatID, userID int) error {
	role, err := s.memberRole(ctx, chatID, userID)
	if err != nil {
		return err
	}

	if role != "admin" {
		return fmt.Errorf(
			"role invalid: %w",
			ErrForbidden,
		)
	}
	return nil
}
