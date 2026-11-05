package messages

import (
	"fmt"
	"path/filepath"
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

func validateCaption(caption string) (string, error) {
	caption = strings.TrimSpace(caption)
	if utf8.RuneCountInString(caption) > 500 {
		return "", fmt.Errorf(
			"caption length must be 0-500: %w",
			ErrInvalidInput,
		)
	}
	return caption, nil
}

func cleanFileName(name string) string {
	name = strings.TrimSpace(filepath.Base(strings.ReplaceAll(name, "\\", "/")))
	if name == "" || name == "." || name == "/" {
		return "file"
	}
	if runes := []rune(name); len(runes) > 100 {
		name = string(runes[len(runes)-100:])
	}
	return name
}
