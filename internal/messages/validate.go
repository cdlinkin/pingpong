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
