package messages

import (
	"context"
	"fmt"
)

func (r *Repository) SearchMessages(
	ctx context.Context,
	chatID int,
	query string,
) ([]MessageModel, error) {
	querySearch := `
	SELECT m.id, m.chat_id, m.user_id, m.message, m.created_at
	FROM messages AS m
	WHERE m.chat_id = $1 AND m.message ILIKE '%' || $2 || '%' AND m.deleted_at IS NULL
	ORDER BY m.id DESC
	LIMIT 50;
	`

	rows, err := r.pool.Query(ctx, querySearch, chatID, query)
	if err != nil {
		return nil, fmt.Errorf(
			"search message: %w",
			err,
		)
	}
	defer rows.Close()

	var messages []MessageModel
	for rows.Next() {
		var m MessageModel
		if err := rows.Scan(
			&m.ID,
			&m.ChatID,
			&m.UserID,
			&m.Message,
			&m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"scan message: %w",
				err,
			)
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"rows: %w",
			err,
		)
	}

	return messages, nil
}
