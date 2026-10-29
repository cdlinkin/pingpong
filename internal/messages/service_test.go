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

	gotText     string
	updateCalls int
	deleteCalls int
	updateErr   error
	deleteErr   error

	markCalls  int
	gotMarkID  int
	lastReadID int
	markErr    error
	readsCalls int
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

func (f *fakeRepo) UpdateMessage(ctx context.Context, chatID, msgID, userID int, msg string) (MessageModel, error) {
	f.updateCalls++
	f.gotText = msg
	if f.updateErr != nil {
		return MessageModel{}, f.updateErr
	}
	return MessageModel{ID: msgID, ChatID: chatID, UserID: userID, Message: msg}, nil
}

func (f *fakeRepo) DeleteMessage(ctx context.Context, chatID, msgID, userID int) error {
	f.deleteCalls++
	return f.deleteErr
}

func (f *fakeRepo) ListReads(ctx context.Context, chatID int) ([]ReadModel, error) {
	f.readsCalls++
	return []ReadModel{{UserID: member, LastReadID: 7}}, nil
}

func (f *fakeRepo) MarkRead(ctx context.Context, chatID, userID, msgID int) (int, error) {
	f.markCalls++
	f.gotMarkID = msgID
	if f.markErr != nil {
		return 0, f.markErr
	}
	return f.lastReadID, nil
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

func TestEditMessage(t *testing.T) {
	ctx := context.Background()

	t.Run("ok, text is trimmed", func(t *testing.T) {
		repo := newFakeRepo()
		msg, err := NewService(repo).EditMessage(ctx, member, chatID, 5, "  новый текст ")
		if err != nil {
			t.Fatalf("edit: %v", err)
		}
		if repo.gotText != "новый текст" || msg.Message != "новый текст" {
			t.Fatalf("text must be trimmed, got %q", repo.gotText)
		}
	})

	tests := []struct {
		name      string
		userID    int
		text      string
		updateErr error
		wantErr   error
		wantCalls int
	}{
		{"empty text", member, "   ", nil, ErrInvalidInput, 0},
		{"not a member", stranger, "hi", nil, ErrChatNotFound, 0},
		{"not author or deleted", member, "hi", ErrMessageNotFound, ErrMessageNotFound, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			repo.updateErr = tt.updateErr
			_, err := NewService(repo).EditMessage(ctx, tt.userID, chatID, 5, tt.text)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("want %v, got %v", tt.wantErr, err)
			}
			if repo.updateCalls != tt.wantCalls {
				t.Fatalf("want %d update calls, got %d", tt.wantCalls, repo.updateCalls)
			}
		})
	}
}

func TestDeleteMessage(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		userID    int
		deleteErr error
		wantErr   error
		wantCalls int
	}{
		{"ok", member, nil, nil, 1},
		{"not a member", stranger, nil, ErrChatNotFound, 0},
		{"not author or already deleted", member, ErrMessageNotFound, ErrMessageNotFound, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			repo.deleteErr = tt.deleteErr
			err := NewService(repo).DeleteMessage(ctx, tt.userID, chatID, 5)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("want %v, got %v", tt.wantErr, err)
			}
			if repo.deleteCalls != tt.wantCalls {
				t.Fatalf("want %d delete calls, got %d", tt.wantCalls, repo.deleteCalls)
			}
		})
	}
}

func TestMarkRead(t *testing.T) {
	ctx := context.Background()

	t.Run("ok returns final mark from repository", func(t *testing.T) {
		repo := newFakeRepo()
		repo.lastReadID = 120
		got, err := NewService(repo).MarkRead(ctx, chatID, member, 80)
		if err != nil {
			t.Fatalf("mark read: %v", err)
		}
		if got != 120 {
			t.Fatalf("want 120, got %d", got)
		}
		if repo.gotMarkID != 80 {
			t.Fatalf("want message id 80 passed to repository, got %d", repo.gotMarkID)
		}
	})

	tests := []struct {
		name      string
		msgID     int
		markErr   error
		wantErr   error
		wantCalls int
	}{
		{"zero id", 0, nil, ErrInvalidInput, 0},
		{"negative id", -5, nil, ErrInvalidInput, 0},
		{"not a member", 10, ErrChatNotFound, ErrChatNotFound, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			repo.markErr = tt.markErr
			_, err := NewService(repo).MarkRead(ctx, chatID, member, tt.msgID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("want %v, got %v", tt.wantErr, err)
			}
			if repo.markCalls != tt.wantCalls {
				t.Fatalf("want %d repository calls, got %d", tt.wantCalls, repo.markCalls)
			}
		})
	}
}

func TestListReads(t *testing.T) {
	ctx := context.Background()

	t.Run("member gets marks", func(t *testing.T) {
		repo := newFakeRepo()
		reads, err := NewService(repo).ListReads(ctx, member, chatID)
		if err != nil {
			t.Fatalf("list reads: %v", err)
		}
		if len(reads) != 1 || reads[0].LastReadID != 7 {
			t.Fatalf("unexpected reads: %+v", reads)
		}
	})

	t.Run("stranger sees not found", func(t *testing.T) {
		repo := newFakeRepo()
		_, err := NewService(repo).ListReads(ctx, stranger, chatID)
		if !errors.Is(err, ErrChatNotFound) {
			t.Fatalf("want ErrChatNotFound, got %v", err)
		}
		if repo.readsCalls != 0 {
			t.Fatal("marks must not be loaded for a stranger")
		}
	})
}
