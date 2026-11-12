package chats

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (r *Repository) CreateGroupChat(
	ctx context.Context,
	title string,
	creatorID int,
) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf(
			"begin tx: %w",
			err,
		)
	}
	defer tx.Rollback(ctx)

	var chatID int
	err = tx.QueryRow(
		ctx,
		`INSERT INTO chats (title, type) 
		VALUES ($1, 'group') 
		RETURNING id;`,
		title,
	).Scan(&chatID)
	if err != nil {
		return 0, fmt.Errorf(
			"insert chat: %w",
			err,
		)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO chat_members (chat_id, user_id, role)
		VALUES ($1, $2, 'admin');`,
		chatID, creatorID,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"insert chat member: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf(
			"commit tx: %w",
			err,
		)
	}

	return chatID, nil
}

func (r *Repository) CreateDirectChat(
	ctx context.Context,
	userA, userB int,
) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf(
			"begin tx: %w",
			err,
		)
	}
	defer tx.Rollback(ctx)

	var chatID int
	err = tx.QueryRow(
		ctx,
		`INSERT INTO chats (type)
		VALUES ('direct') RETURNING id;`,
	).Scan(&chatID)
	if err != nil {
		return 0, fmt.Errorf(
			"insert chat: %w",
			err,
		)
	}

	_, err = tx.Exec(
		ctx,
		`INSERT INTO chat_members (chat_id, user_id, role)
		VALUES ($1, $2, 'member'), ($1, $3, 'member');`,
		chatID, userA, userB,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return 0, ErrUserNotFound
		}
		return 0, fmt.Errorf(
			"insert chat members: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf(
			"commit tx: %w",
			err,
		)
	}

	return chatID, nil
}

func (r *Repository) FindDirectChat(
	ctx context.Context,
	userA, userB int,
) (int, error) {
	var chatID int
	err := r.pool.QueryRow(
		ctx,
		`SELECT c.id
		FROM chats c
		JOIN chat_members a ON a.chat_id = c.id AND a.user_id = $1
		JOIN chat_members b ON b.chat_id = c.id AND b.user_id = $2
		WHERE c.type = 'direct';`,
		userA, userB,
	).Scan(&chatID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrChatNotFound
		}
		return 0, fmt.Errorf(
			"find direct chat: %w",
			err,
		)
	}

	return chatID, nil
}

func (r *Repository) ListUserChats(
	ctx context.Context,
	userID int,
) ([]ChatModel, error) {
	query := `
	SELECT c.id, c.title, c.type, c.created_at, m.muted,
	(SELECT COUNT(*)
	FROM messages msg
	WHERE msg.chat_id = c.id
	AND msg.id > m.last_read_message_id
	AND msg.user_id <> $1
	AND msg.deleted_at IS NULL) AS unread_count,
	(SELECT u.login
	FROM chat_members pm
	JOIN users u ON u.id = pm.user_id
	WHERE pm.chat_id = c.id
	AND pm.user_id <> $1
	AND c.type = 'direct'
	LIMIT 1) AS peer_login,
	(SELECT COUNT(*)
	FROM messages msg
	JOIN message_mentions mm ON mm.message_id = msg.id AND mm.user_id = $1
	WHERE msg.chat_id = c.id
	AND msg.id > m.last_read_message_id
	AND msg.deleted_at IS NULL) AS unread_mentions
	FROM chats c
	JOIN chat_members m ON m.chat_id = c.id
	WHERE m.user_id = $1
	ORDER BY c.created_at DESC;
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf(
			"list user chats: %w",
			err,
		)
	}
	defer rows.Close()

	var chats []ChatModel
	for rows.Next() {
		var c ChatModel
		if err := rows.Scan(
			&c.ID,
			&c.Title,
			&c.Type,
			&c.CreatedAt,
			&c.Muted,
			&c.UnreadCount,
			&c.PeerLogin,
			&c.UnreadMentions,
		); err != nil {
			return nil, fmt.Errorf(
				"scan chat: %w",
				err,
			)
		}
		chats = append(chats, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"rows: %w",
			err,
		)
	}

	return chats, nil
}
