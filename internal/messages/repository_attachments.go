package messages

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const attachmentColumns = `
	a.id, a.message_id, m.chat_id, a.file_name, a.content_type,
	a.size, a.storage_key, a.created_at, m.deleted_at IS NOT NULL
`

func (r *Repository) CreateAttachmentMessage(
	ctx context.Context,
	chatID, userID int,
	caption string,
	replyToID *int,
	a AttachmentModel,
) (MessageModel, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return MessageModel{}, fmt.Errorf(
			"begin tx: %w",
			err,
		)
	}
	defer tx.Rollback(ctx)

	query := `
	WITH ins AS (
    INSERT INTO messages (chat_id, user_id, message, reply_to_id)
    VALUES ($1, $2, $3, $4)
    RETURNING id, chat_id, user_id, message, created_at, reply_to_id
	)
	SELECT ins.id, ins.chat_id, ins.user_id, ins.message, ins.created_at,
	ins.reply_to_id, r.user_id,
	CASE WHEN r.deleted_at IS NULL THEN r.message ELSE '' END,
	ra.file_name
	FROM ins
	LEFT JOIN messages r ON r.id = ins.reply_to_id
	LEFT JOIN message_attachments ra ON ra.message_id = r.id AND r.deleted_at IS NULL;
	`

	var m MessageModel
	if err := tx.QueryRow(ctx, query, chatID, userID, caption, replyToID).Scan(
		&m.ID,
		&m.ChatID,
		&m.UserID,
		&m.Message,
		&m.CreatedAt,
		&m.ReplyToID,
		&m.ReplyUserID,
		&m.ReplyMessage,
		&m.ReplyFileName,
	); err != nil {
		return MessageModel{}, fmt.Errorf(
			"insert message: %w",
			err,
		)
	}

	a.MsgID = m.ID
	a.ChatID = m.ChatID
	if err := tx.QueryRow(
		ctx,
		`INSERT INTO message_attachments (message_id, file_name, content_type, size, storage_key)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at;`,
		a.MsgID,
		a.FileName,
		a.ContentType,
		a.Size,
		a.StorageKey,
	).Scan(&a.ID, &a.CreatedAt); err != nil {
		return MessageModel{}, fmt.Errorf(
			"insert attachment: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return MessageModel{}, fmt.Errorf(
			"commit tx: %w",
			err,
		)
	}

	m.Attachment = &a
	return m, nil
}

func (r *Repository) ListAttachments(
	ctx context.Context,
	msgIDs []int,
) ([]AttachmentModel, error) {
	query := `SELECT` + attachmentColumns + `
	FROM message_attachments a
	JOIN messages m ON m.id = a.message_id
	WHERE a.message_id = ANY($1);
	`

	rows, err := r.pool.Query(ctx, query, msgIDs)
	if err != nil {
		return nil, fmt.Errorf(
			"list attachments: %w",
			err,
		)
	}

	attachments, err := pgx.CollectRows(rows, pgx.RowToStructByPos[AttachmentModel])
	if err != nil {
		return nil, fmt.Errorf(
			"collect attachments: %w",
			err,
		)
	}

	return attachments, nil
}

func (r *Repository) GetAttachment(
	ctx context.Context,
	id int,
) (AttachmentModel, error) {
	query := `SELECT` + attachmentColumns + `
	FROM message_attachments a
	JOIN messages m ON m.id = a.message_id
	WHERE a.id = $1;
	`

	rows, err := r.pool.Query(ctx, query, id)
	if err != nil {
		return AttachmentModel{}, fmt.Errorf(
			"get attachment: %w",
			err,
		)
	}

	a, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[AttachmentModel])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AttachmentModel{}, ErrAttachmentNotFound
		}
		return AttachmentModel{}, fmt.Errorf(
			"collect attachment: %w",
			err,
		)
	}

	return a, nil
}
