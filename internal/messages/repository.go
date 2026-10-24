package messages

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

func (r *Repository) IsMember(
	ctx context.Context,
	chatID, userID int,
) (bool, error) {
	query := `
	SELECT EXISTS 
	(SELECT 1 
	FROM chat_members 
	WHERE chat_id = $1 AND user_id = $2);	
	`
	var ok bool
	err := r.pool.QueryRow(
		ctx,
		query,
		chatID,
		userID,
	).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf(
			"is member: %w",
			err,
		)
	}
	return ok, nil
}

func (r *Repository) CreateMessage(
	ctx context.Context,
	chatID, userID int,
	message string,
) (MessageModel, error) {
	query := `
	INSERT INTO messages (chat_id, user_id, message)
	VALUES ($1,$2,$3)
	RETURNING id, chat_id, user_id, message, created_at;
	`

	row := r.pool.QueryRow(ctx, query, chatID, userID, message)

	var messageModel MessageModel

	if err := row.Scan(
		&messageModel.ID,
		&messageModel.ChatID,
		&messageModel.UserID,
		&messageModel.Message,
		&messageModel.CreatedAt,
	); err != nil {
		return MessageModel{}, fmt.Errorf(
			"create message: %w",
			err,
		)
	}
	return messageModel, nil
}
