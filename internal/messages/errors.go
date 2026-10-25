package messages

import "errors"

var (
	ErrChatNotFound = errors.New("chat not found")
	ErrInvalidInput = errors.New("invalid input")
)
