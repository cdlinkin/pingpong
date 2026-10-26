package messages

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

func validateMsg(msg string) (string, error) {
	msg = strings.TrimSpace(msg)
	length := utf8.RuneCountInString(msg)

	if length < 1 || length > 500 {
		return "", fmt.Errorf(
			"message length must be 1-500: %w",
			ErrInvalidInput,
		)
	}

	return msg, nil
}

func validateQuery(q string) (string, error) {
	q = strings.TrimSpace(q)
	if utf8.RuneCountInString(q) == 0 || utf8.RuneCountInString(q) > 100 {
		return "", fmt.Errorf(
			"query: %w",
			ErrInvalidInput,
		)
	}
	return q, nil
}
