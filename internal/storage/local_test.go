package storage

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func TestLocalSaveOpenRemove(t *testing.T) {
	s, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	key, size, err := s.Save(strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if size != 5 || len(key) != 32 {
		t.Fatalf("unexpected key %q or size %d", key, size)
	}

	f, err := s.Open(key)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	data, _ := io.ReadAll(f)
	f.Close()
	if string(data) != "hello" {
		t.Fatalf("want hello, got %q", data)
	}

	if err := s.Remove(key); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := s.Open(key); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want not exist, got %v", err)
	}
}

func TestLocalRejectsBadKeys(t *testing.T) {
	s, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	for _, key := range []string{"", "../../etc/passwd", "zz" + strings.Repeat("0", 30)} {
		if _, err := s.Open(key); !errors.Is(err, ErrInvalidKey) {
			t.Fatalf("key %q: want ErrInvalidKey, got %v", key, err)
		}
	}
}
