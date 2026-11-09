package messages

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) AddMentions(
	ctx context.Context,
	msgID, chatID, authorID int,
	logins []string,
) ([]int, error) {
	query := `
	INSERT INTO message_mentions (message_id, user_id)
	SELECT $1, u.id
	FROM users u
	JOIN chat_members m ON m.user_id = u.id AND m.chat_id = $2
	WHERE u.login = ANY($4)
	AND u.id <> $3
	ON CONFLICT DO NOTHING
	RETURNING user_id;
	`

	rows, err := r.pool.Query(ctx, query, msgID, chatID, authorID, logins)
	if err != nil {
		return nil, fmt.Errorf(
			"add mentions: %w",
			err,
		)
	}

	ids, err := pgx.CollectRows(rows, pgx.RowTo[int])
	if err != nil {
		return nil, fmt.Errorf(
			"collect mentions: %w",
			err,
		)
	}

	return ids, nil
}
