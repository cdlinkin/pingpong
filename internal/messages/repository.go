package messages

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
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

func (r *Repository) ListMessage(
	ctx context.Context,
	chatID int,
	beforeID, limit int,
) ([]MessageModel, error) {
	query := `
	SELECT m.id, m.chat_id, m.user_id, 
	CASE 
	WHEN m.deleted_at 
	IS NULL THEN m.message 
	ELSE '' END AS message, m.created_at, m.edited_at, m.deleted_at
	FROM messages AS m
	WHERE m.chat_id = $1 AND m.id < $2
	ORDER BY m.id DESC
	LIMIT $3;
	`

	rows, err := r.pool.Query(ctx, query, chatID, beforeID, limit)
	if err != nil {
		return nil, fmt.Errorf(
			"list message: %w",
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
			&m.EditedAt,
			&m.DeletedAt,
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

func (r *Repository) ListMemberIDs(
	ctx context.Context,
	chatID int,
) ([]int, error) {
	query := `
	SELECT user_id
	FROM chat_members
	WHERE chat_id = $1;
	`

	rows, err := r.pool.Query(ctx, query, chatID)
	if err != nil {
		return nil, fmt.Errorf(
			"list member ids: %w",
			err,
		)
	}
	defer rows.Close()

	var userID []int
	for rows.Next() {
		var uid int
		if err := rows.Scan(
			&uid,
		); err != nil {
			return nil, fmt.Errorf(
				"scan member id: %w",
				err,
			)
		}
		userID = append(userID, uid)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"rows: %w",
			err,
		)
	}

	return userID, nil
}

func (r *Repository) UpdateMessage(
	ctx context.Context,
	chatID, msgID, userID int,
	msg string,
) (MessageModel, error) {
	query := `
	UPDATE messages
	SET message = $4, edited_at = now()
	WHERE id = $2
	AND chat_id = $1
    AND user_id = $3
	AND deleted_at IS NULL
	RETURNING id, chat_id, user_id, message, created_at, edited_at, deleted_at;
	`

	var m MessageModel
	if err := r.pool.QueryRow(ctx, query, chatID, msgID, userID, msg).Scan(
		&m.ID,
		&m.ChatID,
		&m.UserID,
		&m.Message,
		&m.CreatedAt,
		&m.EditedAt,
		&m.DeletedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MessageModel{}, ErrMessageNotFound
		}
		return MessageModel{}, fmt.Errorf(
			"scan update message: %w",
			err,
		)
	}

	return m, nil
}

func (r *Repository) DeleteMessage(
	ctx context.Context,
	chatID, msgID, userID int,
) error {
	query := `
	UPDATE messages
	SET deleted_at = now()
	WHERE id = $2
	AND chat_id = $1
    AND user_id = $3
	AND deleted_at IS NULL
	RETURNING id;
	`

	var id int
	if err := r.pool.QueryRow(ctx, query, chatID, msgID, userID).Scan(
		&id,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMessageNotFound
		}
		return fmt.Errorf(
			"scan delete message: %w",
			err,
		)
	}

	return nil
}
