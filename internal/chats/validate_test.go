package chats

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateTitle(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		want    string
		wantErr bool
	}{
		{"ok", "Go team", "Go team", false},
		{"trimmed", "  Go team  ", "Go team", false},
		{"max 50 runes", strings.Repeat("я", 50), strings.Repeat("я", 50), false},
		{"empty", "", "", true},
		{"only spaces", "   ", "", true},
		{"too long", strings.Repeat("a", 51), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateTitle(tt.title)
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
