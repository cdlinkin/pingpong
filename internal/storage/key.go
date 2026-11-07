package storage

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func newKey() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf(
			"generate key: %w",
			err,
		)
	}
	return hex.EncodeToString(buf), nil
}

func checkKey(key string) error {
	if len(key) != 32 {
		return ErrInvalidKey
	}
	if _, err := hex.DecodeString(key); err != nil {
		return ErrInvalidKey
	}
	return nil
}
