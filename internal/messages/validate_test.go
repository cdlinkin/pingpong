package messages

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateMsg(t *testing.T) {
	tests := []struct {
		name    string
		msg     string
		want    string
		wantErr bool
	}{
		{"ok", "привет", "привет", false},
		{"trimmed", "  hi \n", "hi", false},
		{"max 500 runes", strings.Repeat("я", 500), strings.Repeat("я", 500), false},
		{"empty", "", "", true},
		{"only spaces", " \t\n ", "", true},
		{"too long", strings.Repeat("a", 501), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateMsg(tt.msg)
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

func TestValidateQuery(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		want    string
		wantErr bool
	}{
		{"ok", "go", "go", false},
		{"trimmed", "  go ", "go", false},
		{"max 100 runes", strings.Repeat("я", 100), strings.Repeat("я", 100), false},
		{"empty", "", "", true},
		{"only spaces", "   ", "", true},
		{"too long", strings.Repeat("a", 101), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateQuery(tt.query)
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
