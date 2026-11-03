package messages

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) MarkRead(
	ctx context.Context,
	chatID, userID, msgID int,
) (int, error) {
	query := `
	UPDATE chat_members 
	SET last_read_message_id = GREATEST(last_read_message_id, 
	LEAST($3, (
	SELECT COALESCE(MAX(id), 0) 
	FROM messages 
	WHERE chat_id = $1))) 
	WHERE chat_id = $1
	AND user_id = $2
	RETURNING last_read_message_id;
	`

	var lastReadID int
	if err := r.pool.QueryRow(
		ctx,
		query,
		chatID,
		userID,
		msgID,
	).Scan(&lastReadID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrChatNotFound
		}
		return 0, fmt.Errorf(
			"scan mark read message: %w",
			err,
		)
	}

	return lastReadID, nil
}

func (r *Repository) ListReads(
	ctx context.Context,
	chatID int,
) ([]ReadModel, error) {
	query := `
	SELECT user_id, last_read_message_id
	FROM chat_members
	WHERE chat_id = $1;
	`

	rows, err := r.pool.Query(ctx, query, chatID)
	if err != nil {
		return nil, fmt.Errorf(
			"list reads: %w",
			err,
		)
	}
	defer rows.Close()

	var reads []ReadModel
	for rows.Next() {
		var rm ReadModel
		if err := rows.Scan(
			&rm.UserID,
			&rm.LastReadID,
		); err != nil {
			return nil, fmt.Errorf(
				"scan read: %w",
				err,
			)
		}
		reads = append(reads, rm)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"rows: %w",
			err,
		)
	}

	return reads, nil
}
