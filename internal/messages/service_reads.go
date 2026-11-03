package messages

import (
	"context"
	"fmt"
)

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
	if err := s.ensureMember(ctx, chatID, userID); err != nil {
		return nil, err
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
