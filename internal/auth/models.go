package auth

import "time"

type UserModel struct {
	ID           int
	Login        string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}
