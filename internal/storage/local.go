package storage

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var ErrInvalidKey = errors.New("invalid storage key")

type Local struct {
	dir string
}

func NewLocal(dir string) (*Local, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf(
			"create upload dir: %w",
			err,
		)
	}
	return &Local{
		dir: dir,
	}, nil
}

func (l *Local) Save(r io.Reader) (string, int64, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", 0, fmt.Errorf(
			"generate key: %w",
			err,
		)
	}
	key := hex.EncodeToString(buf)

	f, err := os.OpenFile(filepath.Join(l.dir, key), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", 0, fmt.Errorf(
			"create file: %w",
			err,
		)
	}

	size, err := io.Copy(f, r)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(f.Name())
		return "", 0, fmt.Errorf(
			"write file: %w",
			err,
		)
	}

	return key, size, nil
}

func (l *Local) Open(key string) (*os.File, error) {
	path, err := l.path(key)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

func (l *Local) Remove(key string) error {
	path, err := l.path(key)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

func (l *Local) path(key string) (string, error) {
	if len(key) != 32 {
		return "", ErrInvalidKey
	}
	if _, err := hex.DecodeString(key); err != nil {
		return "", ErrInvalidKey
	}
	return filepath.Join(l.dir, key), nil
}
