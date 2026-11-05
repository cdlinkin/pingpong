package messages

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeStorage struct {
	dir   string
	files map[string][]byte
	saved int
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{files: map[string][]byte{}}
}

func (s *fakeStorage) Save(r io.Reader) (string, int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", 0, err
	}
	s.saved++
	key := fmt.Sprintf("key%d", s.saved)
	s.files[key] = data
	return key, int64(len(data)), nil
}

func (s *fakeStorage) Open(key string) (*os.File, error) {
	data, ok := s.files[key]
	if !ok {
		return nil, os.ErrNotExist
	}
	path := filepath.Join(s.dir, key)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return nil, err
	}
	return os.Open(path)
}

func (s *fakeStorage) Remove(key string) error {
	delete(s.files, key)
	return nil
}

func (f *fakeRepo) CreateAttachmentMessage(
	ctx context.Context,
	chatID, userID int,
	caption string,
	replyToID *int,
	a AttachmentModel,
) (MessageModel, error) {
	m, _ := f.CreateMessage(ctx, chatID, userID, caption, replyToID)
	a.ID = len(f.attachments) + 1
	a.MsgID = m.ID
	a.ChatID = chatID
	f.attachments = append(f.attachments, a)
	m.Attachment = &a
	return m, nil
}

func (f *fakeRepo) ListAttachments(ctx context.Context, msgIDs []int) ([]AttachmentModel, error) {
	var out []AttachmentModel
	for _, a := range f.attachments {
		for _, id := range msgIDs {
			if a.MsgID == id {
				out = append(out, a)
			}
		}
	}
	return out, nil
}

func (f *fakeRepo) GetAttachment(ctx context.Context, id int) (AttachmentModel, error) {
	for _, a := range f.attachments {
		if a.ID == id {
			return a, nil
		}
	}
	return AttachmentModel{}, ErrAttachmentNotFound
}

var pngHeader = []byte("\x89PNG\r\n\x1a\n0000IHDR")

func TestSendAttachment(t *testing.T) {
	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		repo := newFakeRepo()
		store := newFakeStorage()
		msg, err := NewService(repo, store).SendAttachment(
			ctx,
			chatID,
			member,
			"  схема  ",
			nil,
			`C:\Users\me\schema.png`,
			bytes.NewReader(pngHeader),
		)
		if err != nil {
			t.Fatalf("send: %v", err)
		}
		a := msg.Attachment
		if a == nil || a.FileName != "schema.png" || a.ContentType != "image/png" || a.Size != int64(len(pngHeader)) {
			t.Fatalf("unexpected attachment: %+v", a)
		}
		if msg.Message != "схема" {
			t.Fatalf("caption not trimmed: %q", msg.Message)
		}
		if len(store.files) != 1 {
			t.Fatalf("want 1 stored file, got %d", len(store.files))
		}
	})

	tests := []struct {
		name    string
		userID  int
		caption string
		replyTo *int
		content []byte
		wantErr error
	}{
		{"stranger", stranger, "", nil, pngHeader, ErrChatNotFound},
		{"empty file", member, "", nil, nil, ErrEmptyFile},
		{"long caption", member, strings.Repeat("я", 501), nil, pngHeader, ErrInvalidInput},
		{"unknown reply", member, "", intPtr(99), pngHeader, ErrMessageNotFound},
		{"executable", member, "", nil, []byte("MZ\x90\x00\x03\x00\x00\x00\x04\x00\x00\x00\xff\xff"), ErrUnsupportedFile},
		{"html", member, "", nil, []byte("<html><script>alert(1)</script></html>"), ErrUnsupportedFile},
		{"too large", member, "", nil, append(pngHeader, make([]byte, MaxAttachmentSize)...), ErrFileTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			store := newFakeStorage()
			_, err := NewService(repo, store).SendAttachment(
				ctx,
				chatID,
				tt.userID,
				tt.caption,
				tt.replyTo,
				"file.bin",
				bytes.NewReader(tt.content),
			)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("want %v, got %v", tt.wantErr, err)
			}
			if len(repo.attachments) != 0 || len(store.files) != 0 {
				t.Fatalf("nothing must be saved: repo=%d files=%d", len(repo.attachments), len(store.files))
			}
		})
	}
}

func TestGetAttachment(t *testing.T) {
	ctx := context.Background()

	setup := func(t *testing.T) (*Service, *fakeRepo) {
		repo := newFakeRepo()
		store := newFakeStorage()
		store.dir = t.TempDir()
		svc := NewService(repo, store)
		if _, err := svc.SendAttachment(ctx, chatID, member, "", nil, "a.png", bytes.NewReader(pngHeader)); err != nil {
			t.Fatalf("send: %v", err)
		}
		return svc, repo
	}

	t.Run("member gets file", func(t *testing.T) {
		svc, _ := setup(t)
		a, f, err := svc.GetAttachment(ctx, 3, 1)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		defer f.Close()
		data, _ := io.ReadAll(f)
		if a.FileName != "a.png" || !bytes.Equal(data, pngHeader) {
			t.Fatalf("unexpected attachment %+v / %q", a, data)
		}
	})

	t.Run("stranger gets not found", func(t *testing.T) {
		svc, _ := setup(t)
		if _, _, err := svc.GetAttachment(ctx, stranger, 1); !errors.Is(err, ErrAttachmentNotFound) {
			t.Fatalf("want ErrAttachmentNotFound, got %v", err)
		}
	})

	t.Run("deleted message", func(t *testing.T) {
		svc, repo := setup(t)
		repo.attachments[0].Deleted = true
		if _, _, err := svc.GetAttachment(ctx, member, 1); !errors.Is(err, ErrAttachmentNotFound) {
			t.Fatalf("want ErrAttachmentNotFound, got %v", err)
		}
	})

	t.Run("unknown id", func(t *testing.T) {
		svc, _ := setup(t)
		if _, _, err := svc.GetAttachment(ctx, member, 42); !errors.Is(err, ErrAttachmentNotFound) {
			t.Fatalf("want ErrAttachmentNotFound, got %v", err)
		}
	})
}

func TestListMessageAttachments(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	now := time.Now()
	repo.listed = []MessageModel{{ID: 3}, {ID: 2, DeletedAt: &now}, {ID: 1}}
	repo.attachments = []AttachmentModel{
		{ID: 1, MsgID: 3, FileName: "a.png"},
		{ID: 2, MsgID: 2, FileName: "b.pdf"},
	}

	msgs, err := NewService(repo, newFakeStorage()).ListMessage(ctx, member, chatID, 0, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if msgs[0].Attachment == nil || msgs[0].Attachment.FileName != "a.png" {
		t.Fatalf("message 3: want a.png, got %+v", msgs[0].Attachment)
	}
	if msgs[1].Attachment != nil {
		t.Fatalf("deleted message must have no attachment, got %+v", msgs[1].Attachment)
	}
	if msgs[2].Attachment != nil {
		t.Fatalf("message 1: want none, got %+v", msgs[2].Attachment)
	}
}

func intPtr(v int) *int {
	return &v
}
