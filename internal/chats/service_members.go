package chats

import (
	"context"
	"fmt"
)

func (s *Service) AddMember(
	ctx context.Context,
	requesterID, chatID, newUserID int,
) error {
	if err := s.ensureAdmin(ctx, chatID, requesterID); err != nil {
		return err
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
	if err := s.ensureAdmin(ctx, chatID, requesterID); err != nil {
		return err
	}

	if err := s.repository.RemoveMember(ctx, chatID, targetID); err != nil {
		return fmt.Errorf(
			"remove member: %w",
			err,
		)
	}
	return nil
}

func (s *Service) ListMembers(
	ctx context.Context,
	userID, chatID int,
) ([]MemberModel, error) {
	if _, err := s.memberRole(ctx, chatID, userID); err != nil {
		return nil, err
	}

	members, err := s.repository.ListMembers(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf(
			"list members: %w",
			err,
		)
	}

	return members, nil
}
