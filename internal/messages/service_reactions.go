package messages

import (
	"context"
	"fmt"
)

var allowedEmoji = map[string]bool{
	"👍":  true,
	"❤️": true,
	"😂":  true,
	"😮":  true,
	"😢":  true,
	"🔥":  true,
}

func (s *Service) ToggleReaction(
	ctx context.Context,
	userID, chatID, msgID int,
	emoji string,
) ([]ReactionModel, error) {
	if !allowedEmoji[emoji] {
		return nil, fmt.Errorf(
			"not allowed emoji: %w",
			ErrInvalidInput,
		)
	}

	if err := s.ensureMember(ctx, chatID, userID); err != nil {
		return nil, err
	}

	ok, err := s.repository.MessageInChat(ctx, chatID, msgID)
	if err != nil {
		return nil, fmt.Errorf(
			"message in chat: %w",
			err,
		)
	}

	if !ok {
		return nil, fmt.Errorf(
			"message not in chat: %w",
			ErrMessageNotFound,
		)
	}

	removed, err := s.repository.RemoveReaction(ctx, msgID, userID, emoji)
	if err != nil {
		return nil, fmt.Errorf(
			"remove reaction: %w",
			err,
		)
	}

	if !removed {
		if err := s.repository.UpsertReaction(ctx, msgID, userID, emoji); err != nil {
			return nil, fmt.Errorf(
				"upsert reaction: %w",
				err,
			)
		}
	}

	reactions, err := s.repository.ListReactions(ctx, []int{msgID})
	if err != nil {
		return nil, fmt.Errorf(
			"list reactions: %w",
			err,
		)
	}

	return reactions, nil
}
