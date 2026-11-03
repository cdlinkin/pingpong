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
