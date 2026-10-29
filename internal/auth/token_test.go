package auth

import (
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var testSecret = []byte("test-secret")

func TestCreateAndParseToken(t *testing.T) {
	token, err := createToken(42, testSecret)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	id, err := ParseToken(token, testSecret)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if id != 42 {
		t.Fatalf("want id 42, got %d", id)
	}
}

func TestParseTokenErrors(t *testing.T) {
	signed := func(method jwt.SigningMethod, key any, claims jwt.RegisteredClaims) string {
		t.Helper()
		s, err := jwt.NewWithClaims(method, claims).SignedString(key)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		return s
	}
	valid := jwt.RegisteredClaims{
		Subject:   strconv.Itoa(1),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}

	tests := []struct {
		name  string
		token string
	}{
		{"garbage", "not-a-token"},
		{"wrong secret", signed(jwt.SigningMethodHS256, []byte("other"), valid)},
		{"expired", signed(jwt.SigningMethodHS256, testSecret, jwt.RegisteredClaims{
			Subject:   "1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		})},
		{"other algorithm", signed(jwt.SigningMethodHS512, testSecret, valid)},
		{"subject not a number", signed(jwt.SigningMethodHS256, testSecret, jwt.RegisteredClaims{
			Subject:   "abc",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseToken(tt.token, testSecret); err == nil {
				t.Fatal("want error, got nil")
			}
		})
	}
}
