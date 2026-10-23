package chats

import "errors"

var (
	ErrInvalidInput  = errors.New("invalid input")
	ErrChatNotFound  = errors.New("chat not found")
	ErrUserNotFound  = errors.New("user not found")
	ErrForbidden     = errors.New("forbidden")
	ErrNotMember     = errors.New("not member")
	ErrAlreadyMember = errors.New("already member")
)
