package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func (r *Repository) CreateUser(
	ctx context.Context,
	login, email, passwordHash string,
) (int, error) {
	query := `
	INSERT INTO users (login, email, password_hash)
	VALUES ($1,$2,$3)
	RETURNING id;
	`

	row := r.pool.QueryRow(ctx, query, login, email, passwordHash)

	var userID int

	var pgErr *pgconn.PgError

	if err := row.Scan(
		&userID,
	); err != nil {
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, ErrUserExists
		}
		return 0, fmt.Errorf(
			"create user: %w",
			err,
		)
	}

	return userID, nil
}

func (r *Repository) GetUserByEmail(
	ctx context.Context,
	email string,
) (UserModel, error) {
	query := `
	SELECT id, password_hash
	FROM users
	WHERE email = $1
	`

	row := r.pool.QueryRow(ctx, query, email)

	var user UserModel

	if err := row.Scan(
		&user.ID,
		&user.PasswordHash,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserModel{}, ErrUserNotFound
		}
		return UserModel{}, fmt.Errorf(
			"get user by email: %w",
			err,
		)
	}

	return user, nil
}

func (r *Repository) GetUserByID(
	ctx context.Context,
	id int,
) (UserModel, error) {
	query := `
	SELECT id, login, email, created_at
	FROM users
	WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)

	var user UserModel

	if err := row.Scan(
		&user.ID,
		&user.Login,
		&user.Email,
		&user.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserModel{}, ErrUserNotFound
		}
		return UserModel{}, fmt.Errorf(
			"get user by id: %w",
			err,
		)
	}

	return user, nil
}
