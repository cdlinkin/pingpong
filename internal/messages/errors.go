package messages

import "errors"

var (
	ErrChatNotFound    = errors.New("chat not found")
	ErrInvalidInput    = errors.New("invalid input")
	ErrMessageNotFound = errors.New("message not found")

	ErrAttachmentNotFound = errors.New("attachment not found")
	ErrUnsupportedFile    = errors.New("unsupported file type")
	ErrFileTooLarge       = errors.New("file too large")
	ErrEmptyFile          = errors.New("empty file")
)
