package messages

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"testing"
)

func TestParseMentions(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{"one mention", "привет, @sonya глянь", []string{"sonya"}},
		{"two mentions", "@sonya и @mark, созвон", []string{"sonya", "mark"}},
		{"duplicate", "@sonya @sonya", []string{"sonya"}},
		{"email is not a mention", "пиши на a@b.ru или test@mail.ru", nil},
		{"start of text", "@mark ты где", []string{"mark"}},
		{"dot at the end", "спасибо, @sonya.", []string{"sonya"}},
		{"dot inside login", "@ivan.petrov", []string{"ivan.petrov"}},
		{"too short", "@ab привет", nil},
		{"too short after dot", "@ab.", nil},
		{"cyrillic login", "@соня привет", []string{"соня"}},
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

func TestSendMessageMentions(t *testing.T) {
	ctx := context.Background()

	newRepo := func() *fakeRepo {
		repo := newFakeRepo()
		repo.logins = map[string]int{"me": member, "sonya": 3, "vasya": stranger}
		return repo
	}

	t.Run("only other chat members are mentioned", func(t *testing.T) {
		repo := newRepo()
		msg, err := NewService(repo, newFakeStorage()).SendMessage(ctx, chatID, member, "@sonya @vasya @me @nobody глянь", nil)
		if err != nil {
			t.Fatalf("send: %v", err)
		}
		if !slices.Equal(msg.Mentions, []int{3}) {
			t.Fatalf("want mentions [3], got %v", msg.Mentions)
		}
	})

	t.Run("no @ means no repository call", func(t *testing.T) {
		repo := newRepo()
		msg, err := NewService(repo, newFakeStorage()).SendMessage(ctx, chatID, member, "привет всем", nil)
		if err != nil {
			t.Fatalf("send: %v", err)
		}
		if repo.mentionCalls != 0 || msg.Mentions != nil {
			t.Fatalf("want no mentions, got calls=%d mentions=%v", repo.mentionCalls, msg.Mentions)
		}
	})

	t.Run("repository error is returned", func(t *testing.T) {
		repo := newRepo()
		repo.mentionErr = errors.New("db down")
		if _, err := NewService(repo, newFakeStorage()).SendMessage(ctx, chatID, member, "@sonya", nil); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("caption of attachment", func(t *testing.T) {
		repo := newRepo()
		msg, err := NewService(repo, newFakeStorage()).SendAttachment(ctx, chatID, member, "@sonya схема", nil, "schema.png", bytes.NewReader(pngHeader))
		if err != nil {
			t.Fatalf("send: %v", err)
		}
		if !slices.Equal(msg.Mentions, []int{3}) {
			t.Fatalf("want mentions [3], got %v", msg.Mentions)
		}
	})
}
