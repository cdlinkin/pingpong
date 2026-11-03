package messages

import (
	"context"
	"fmt"
)

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
	if err := s.ensureMember(ctx, chatID, userID); err != nil {
		return nil, err
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
