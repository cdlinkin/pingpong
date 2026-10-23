package chats

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool: pool,
	}
}

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
	err = tx.QueryRow(ctx,
		`INSERT INTO chats (title, type) VALUES ($1, 'group') RETURNING id`,
		title,
	).Scan(&chatID)
	if err != nil {
		return 0, fmt.Errorf("insert chat: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO chat_members (chat_id, user_id, role) VALUES ($1, $2, 'admin')`,
		chatID, creatorID,
	)
	if err != nil {
		return 0, fmt.Errorf("insert chat member: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}

	return chatID, nil
}
