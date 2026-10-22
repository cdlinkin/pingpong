package auth

import "errors"

var (
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidInput       = errors.New("invalid input")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
)
