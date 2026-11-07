package messages

import (
	"context"
	"fmt"
	"math"
)

func (s *Service) SendMessage(
	ctx context.Context,
	chatID, userID int,
	msg string,
	replyToID *int,
) (MessageModel, error) {
	msg, err := validateMsg(msg)
	if err != nil {
		return MessageModel{}, fmt.Errorf(
			"validate message: %w",
			err,
		)
	}

	if err := s.ensureMember(ctx, chatID, userID); err != nil {
		return MessageModel{}, err
	}

	if err := s.ensureReplyTarget(ctx, chatID, replyToID); err != nil {
		return MessageModel{}, err
	}

	msgModel, err := s.repository.CreateMessage(ctx, chatID, userID, msg, replyToID)
	if err != nil {
		return MessageModel{}, fmt.Errorf(
			"create message: %w",
			err,
		)
	}

	return msgModel, nil
}

func (s *Service) ListMessage(
	ctx context.Context,
	userID, chatID int,
	beforeID, limit int,
) ([]MessageModel, error) {
	if err := s.ensureMember(ctx, chatID, userID); err != nil {
		return nil, err
	}

	if beforeID == 0 {
		beforeID = math.MaxInt32
	}

	if limit <= 0 {
		limit = 50
	}
	if limit >= 100 {
		limit = 100
	}
	messages, err := s.repository.ListMessage(ctx, chatID, beforeID, limit)
	if err != nil {
		return nil, fmt.Errorf(
			"list messages: %w",
			err,
		)
	}

	ids := make([]int, 0, len(messages))
	for _, m := range messages {
		ids = append(ids, m.ID)
	}

	reactions, err := s.repository.ListReactions(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list reactions: %w", err)
	}

	byMsg := make(map[int][]ReactionModel)
	for _, r := range reactions {
		byMsg[r.MsgID] = append(byMsg[r.MsgID], r)
	}

	for i := range messages {
		if messages[i].DeletedAt == nil {
			messages[i].Reactions = byMsg[messages[i].ID]
		}
	}

	if err := s.fillAttachments(ctx, messages, ids); err != nil {
		return nil, err
	}

	return messages, nil
}

func (s *Service) EditMessage(
	ctx context.Context,
	userID, chatID, msgID int,
	msg string,
) (MessageModel, error) {
	msg, err := validateMsg(msg)
	if err != nil {
		return MessageModel{}, fmt.Errorf(
			"validate message: %w",
			err,
		)
	}

	if err := s.ensureMember(ctx, chatID, userID); err != nil {
		return MessageModel{}, err
	}

	msgModel, err := s.repository.UpdateMessage(ctx, chatID, msgID, userID, msg)
	if err != nil {
		return MessageModel{}, fmt.Errorf(
			"update message: %w",
			err,
		)
	}

	return msgModel, nil
}

func (s *Service) DeleteMessage(
	ctx context.Context,
	userID, chatID, msgID int,
) error {
	if err := s.ensureMember(ctx, chatID, userID); err != nil {
		return err
	}

	if err := s.repository.DeleteMessage(ctx, chatID, msgID, userID); err != nil {
		return fmt.Errorf(
			"delete message: %w",
			err,
		)
	}

	attachments, err := s.repository.ListAttachments(ctx, []int{msgID})
	if err != nil {
		return fmt.Errorf(
			"list attachments: %w",
			err,
		)
	}

	for _, a := range attachments {
		s.storage.Remove(ctx, a.StorageKey)
	}

	return nil
}
