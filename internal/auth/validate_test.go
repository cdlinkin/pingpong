package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateLogin(t *testing.T) {
	tests := []struct {
		name    string
		login   string
		wantErr bool
	}{
		{"ok", "kuroame", false},
		{"min 3", "abc", false},
		{"max 25", strings.Repeat("a", 25), false},
		{"cyrillic counts runes", "вася", false},
		{"empty", "", true},
		{"too short", "ab", true},
		{"too long", strings.Repeat("a", 26), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLogin(tt.login)
			if tt.wantErr && !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("want nil, got %v", err)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		want    string
		wantErr bool
	}{
		{"ok", "user@mail.ru", "user@mail.ru", false},
		{"trim and lower", "  User@Mail.RU ", "user@mail.ru", false},
		{"empty", "", "", true},
		{"no at", "usermail.ru", "", true},
		{"with name", "Vasya <user@mail.ru>", "", true},
		{"too long", strings.Repeat("a", 245) + "@mail.ru", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateEmail(tt.email)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidInput) {
					t.Fatalf("want ErrInvalidInput, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("want nil, got %v", err)
			}
			if got != tt.want {
				t.Fatalf("want %q, got %q", tt.want, got)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"ok", "password123", false},
		{"min 8", "12345678", false},
		{"max 72 bytes", strings.Repeat("a", 72), false},
		{"too short", "1234567", true},
		{"too long bytes", strings.Repeat("a", 73), true},
		{"cyrillic over 72 bytes", strings.Repeat("я", 37), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePassword(tt.password)
			if tt.wantErr && !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("want nil, got %v", err)
			}
		})
	}
}
