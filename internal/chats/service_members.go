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
