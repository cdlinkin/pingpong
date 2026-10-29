package messages

import (
	"context"
	"errors"
	"math"
	"testing"
)

type fakeRepo struct {
	members map[int][]int

	created     []MessageModel
	gotBeforeID int
	gotLimit    int
	gotQuery    string
	listCalls   int
	searchCalls int
	isMemberErr error
}

func (f *fakeRepo) IsMember(ctx context.Context, chatID, userID int) (bool, error) {
	if f.isMemberErr != nil {
		return false, f.isMemberErr
	}
	for _, id := range f.members[chatID] {
		if id == userID {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeRepo) CreateMessage(ctx context.Context, chatID, userID int, message string) (MessageModel, error) {
	m := MessageModel{ID: len(f.created) + 1, ChatID: chatID, UserID: userID, Message: message}
	f.created = append(f.created, m)
	return m, nil
}

func (f *fakeRepo) ListMessage(ctx context.Context, chatID, beforeID, limit int) ([]MessageModel, error) {
	f.listCalls++
	f.gotBeforeID = beforeID
	f.gotLimit = limit
	return nil, nil
}

func (f *fakeRepo) SearchMessages(ctx context.Context, chatID int, query string) ([]MessageModel, error) {
	f.searchCalls++
	f.gotQuery = query
	return nil, nil
}

func (f *fakeRepo) ListMemberIDs(ctx context.Context, chatID int) ([]int, error) {
	return f.members[chatID], nil
}

const (
	chatID   = 10
	member   = 1
	stranger = 2
)

func newFakeRepo() *fakeRepo {
	return &fakeRepo{members: map[int][]int{chatID: {member, 3}}}
}

func TestSendMessage(t *testing.T) {
	ctx := context.Background()

	t.Run("ok, text is trimmed", func(t *testing.T) {
		repo := newFakeRepo()
		msg, err := NewService(repo).SendMessage(ctx, chatID, member, "  привет  ")
		if err != nil {
			t.Fatalf("send: %v", err)
		}
		if msg.Message != "привет" || msg.ChatID != chatID || msg.UserID != member {
			t.Fatalf("unexpected message: %+v", msg)
		}
	})

	tests := []struct {
		name    string
		userID  int
		chatID  int
		text    string
		wantErr error
	}{
		{"empty text", member, chatID, "   ", ErrInvalidInput},
		{"not a member", stranger, chatID, "hi", ErrChatNotFound},
		{"chat does not exist", member, 999, "hi", ErrChatNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			_, err := NewService(repo).SendMessage(ctx, tt.chatID, tt.userID, tt.text)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("want %v, got %v", tt.wantErr, err)
			}
			if len(repo.created) != 0 {
				t.Fatal("message must not be saved")
			}
		})
	}

	t.Run("repository error is wrapped", func(t *testing.T) {
		dbErr := errors.New("db down")
		repo := newFakeRepo()
		repo.isMemberErr = dbErr
		_, err := NewService(repo).SendMessage(ctx, chatID, member, "hi")
		if !errors.Is(err, dbErr) {
			t.Fatalf("want db error, got %v", err)
		}
	})
}

func TestListMessagePagination(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		beforeID     int
		limit        int
		wantBeforeID int
		wantLimit    int
	}{
		{"defaults", 0, 0, math.MaxInt32, 50},
		{"negative limit", 0, -5, math.MaxInt32, 50},
		{"custom", 120, 20, 120, 20},
		{"limit capped", 0, 1000, math.MaxInt32, 100},
		{"limit exactly 100", 0, 100, math.MaxInt32, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			if _, err := NewService(repo).ListMessage(ctx, member, chatID, tt.beforeID, tt.limit); err != nil {
				t.Fatalf("list: %v", err)
			}
			if repo.gotBeforeID != tt.wantBeforeID {
				t.Fatalf("want beforeID %d, got %d", tt.wantBeforeID, repo.gotBeforeID)
			}
			if repo.gotLimit != tt.wantLimit {
				t.Fatalf("want limit %d, got %d", tt.wantLimit, repo.gotLimit)
			}
		})
	}

	t.Run("not a member", func(t *testing.T) {
		repo := newFakeRepo()
		_, err := NewService(repo).ListMessage(ctx, stranger, chatID, 0, 0)
		if !errors.Is(err, ErrChatNotFound) {
			t.Fatalf("want ErrChatNotFound, got %v", err)
		}
		if repo.listCalls != 0 {
			t.Fatal("messages must not be loaded for a stranger")
		}
	})
}

func TestSearchMessages(t *testing.T) {
	ctx := context.Background()

	t.Run("ok, query is trimmed", func(t *testing.T) {
		repo := newFakeRepo()
		if _, err := NewService(repo).SearchMessages(ctx, member, chatID, "  go "); err != nil {
			t.Fatalf("search: %v", err)
		}
		if repo.gotQuery != "go" {
			t.Fatalf("want query %q, got %q", "go", repo.gotQuery)
		}
	})

	tests := []struct {
		name    string
		userID  int
		query   string
		wantErr error
	}{
		{"empty query", member, "  ", ErrInvalidInput},
		{"not a member", stranger, "go", ErrChatNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			_, err := NewService(repo).SearchMessages(ctx, tt.userID, chatID, tt.query)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("want %v, got %v", tt.wantErr, err)
			}
			if repo.searchCalls != 0 {
				t.Fatal("search must not reach repository")
			}
		})
	}
}
