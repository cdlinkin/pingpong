package chats

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (r *Repository) GetMemberRole(
	ctx context.Context,
	chatID, userID int,
) (string, error) {
	var role string
	err := r.pool.QueryRow(
		ctx,
		`
		SELECT role 
		FROM chat_members 
		WHERE chat_id = $1 AND user_id = $2;
		`,
		chatID,
		userID,
	).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotMember
		}
		return "", fmt.Errorf(
			"get member role: %w",
			err,
		)
	}
	return role, nil
}

func (r *Repository) AddMember(
	ctx context.Context,
	chatID, userID int,
) error {
	query := `
	INSERT INTO chat_members (chat_id, user_id, role) 
	VALUES ($1, $2, 'member');
	`

	_, err := r.pool.Exec(ctx, query, chatID, userID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				return ErrAlreadyMember
			case "23503":
				return ErrUserNotFound
			}
		}
		return fmt.Errorf(
			"add member: %w",
			err,
		)
	}
	return nil
}

func (r *Repository) RemoveMember(
	ctx context.Context,
	chatID, userID int,
) error {
	query := `
	DELETE FROM chat_members 
	WHERE chat_id = $1 AND user_id = $2;
	`

	tag, err := r.pool.Exec(
		ctx,
		query,
		chatID,
		userID,
	)
	if err != nil {
		return fmt.Errorf(
			"remove member: %w",
			err,
		)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotMember
	}
	return nil
}

func (r *Repository) ListMembers(
	ctx context.Context,
	chatID int,
) ([]MemberModel, error) {
	query := `
	SELECT m.user_id, u.login, m.role
	FROM chat_members m
	JOIN users u ON u.id = m.user_id
	WHERE m.chat_id = $1
	ORDER BY u.login;
	`

	rows, err := r.pool.Query(ctx, query, chatID)
	if err != nil {
		return nil, fmt.Errorf(
			"list members: %w",
			err,
		)
	}

	members, err := pgx.CollectRows(rows, pgx.RowToStructByPos[MemberModel])
	if err != nil {
		return nil, fmt.Errorf(
			"collect members: %w",
			err,
		)
	}

	return members, nil
}

func (r *Repository) SetMuted(
	ctx context.Context,
	userID, chatID int,
	muted bool,
) error {
	query := `
	UPDATE chat_members
	SET muted = $3
	WHERE chat_id = $1
	AND user_id = $2;
	`

	tag, err := r.pool.Exec(
		ctx,
		query,
		chatID,
		userID,
		muted,
	)
	if err != nil {
		return fmt.Errorf(
			"set muted: %w",
			err,
		)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotMember
	}

	return nil
}
