package auth

import (
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"
)

func validateLogin(login string) error {
	length := utf8.RuneCountInString(login)
	if length < 3 || length > 25 {
		return fmt.Errorf(
			"login must be 3-25 characters: %w",
			ErrInvalidInput,
		)
	}
	return nil
}

func validateEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	if len(email) > 250 {
		return "", fmt.Errorf(
			"email must be at most 250 characters: %w",
			ErrInvalidInput,
		)
	}

	parsedAddress, err := mail.ParseAddress(email)
	if err != nil || parsedAddress.Address != email {
		return "", fmt.Errorf(
			"invalid email: %w",
			ErrInvalidInput,
		)
	}

	return email, nil
}

func validatePassword(password string) error {
	if utf8.RuneCountInString(password) < 8 {
		return fmt.Errorf(
			"password must be 8 characters: %w",
			ErrInvalidInput,
		)
	}

	if len(password) > 72 {
		return fmt.Errorf(
			"password is too long: %w",
			ErrInvalidInput,
		)
	}
	return nil
}
