package messages

import (
	"context"
	"fmt"
)

func (r *Repository) RemoveReaction(
	ctx context.Context,
	msgID, userID int,
	emoji string,
) (bool, error) {
	query := `
	DELETE FROM message_reactions
	WHERE message_id = $1 AND user_id = $2 AND emoji = $3
	`

	tag, err := r.pool.Exec(ctx, query, msgID, userID, emoji)
	if err != nil {
		return false, fmt.Errorf(
			"remove reaction: %w",
			err,
		)
	}

	return tag.RowsAffected() == 1, nil
}

func (r *Repository) UpsertReaction(
	ctx context.Context,
	msgID, userID int,
	emoji string,
) error {
	query := `
	INSERT INTO message_reactions (message_id, user_id, emoji)
	VALUES ($1, $2, $3)
	ON CONFLICT (message_id, user_id)
	DO UPDATE SET emoji = EXCLUDED.emoji, created_at = now()
	`

	if _, err := r.pool.Exec(ctx, query, msgID, userID, emoji); err != nil {
		return fmt.Errorf(
			"upsert reaction: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) ListReactions(
	ctx context.Context,
	msgIDs []int,
) ([]ReactionModel, error) {
	query := `
	SELECT message_id, user_id, emoji
	FROM message_reactions
	WHERE message_id = ANY($1)
	ORDER BY created_at
	`

	rows, err := r.pool.Query(ctx, query, msgIDs)
	if err != nil {
		return nil, fmt.Errorf(
			"list reaction: %w",
			err,
		)
	}
	defer rows.Close()

	var reactionModels []ReactionModel

	for rows.Next() {
		var r ReactionModel
		if err := rows.Scan(
			&r.MsgID,
			&r.UserID,
			&r.Emoji,
		); err != nil {
			return nil, fmt.Errorf(
				"scan row: %w",
				err,
			)
		}
		reactionModels = append(reactionModels, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"rows error: %w",
			err,
		)
	}
	return reactionModels, nil
}
