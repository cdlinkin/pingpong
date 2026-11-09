package messages

import (
	"slices"
	"testing"
)

func TestParseMentions(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{"one mention", "привет, @robot глянь", []string{"robot"}},
		{"two mentions", "@robot и @mark, созвон", []string{"robot", "mark"}},
		{"duplicate", "@robot @robot", []string{"robot"}},
		{"email is not a mention", "пиши на a@b.ru или test@mail.ru", nil},
		{"start of text", "@mark ты где", []string{"mark"}},
		{"dot at the end", "спасибо, @robot.", []string{"robot"}},
		{"dot inside login", "@ivan.petrov", []string{"ivan.petrov"}},
		{"too short", "@ab привет", nil},
		{"too short after dot", "@ab.", nil},
		{"cyrillic login", "@робот привет", []string{"робот"}},
		{"in brackets", "(@mark)", []string{"mark"}},
		{"no at sign", "просто текст", nil},
		{"lonely at", "@ и всё", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseMentions(tt.text)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("parseMentions(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}
