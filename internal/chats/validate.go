package chats

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

func validateTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	length := utf8.RuneCountInString(title)

	if length < 1 || length > 50 {
		return "", fmt.Errorf(
			"title length must be 1-50: %w",
			ErrInvalidInput,
		)
	}

	return title, nil
}
