package auth

import (
	"encoding/hex"
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

func TestAccessTokenTTL(t *testing.T) {
	token, err := createToken(1, testSecret)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	claims := &jwt.RegisteredClaims{}
	if _, err := jwt.ParseWithClaims(token, claims, func(*jwt.Token) (any, error) { return testSecret, nil }); err != nil {
		t.Fatalf("parse: %v", err)
	}

	left := time.Until(claims.ExpiresAt.Time)
	if left > accessTTL || left < accessTTL-time.Minute {
		t.Fatalf("want token to live about %v, got %v", accessTTL, left)
	}
}

func TestNewRefreshToken(t *testing.T) {
	a, err := newRefreshToken()
	if err != nil {
		t.Fatalf("new refresh token: %v", err)
	}
	b, err := newRefreshToken()
	if err != nil {
		t.Fatalf("new refresh token: %v", err)
	}

	if len(a) != 64 {
		t.Fatalf("want 64 hex chars, got %d", len(a))
	}
	if _, err := hex.DecodeString(a); err != nil {
		t.Fatalf("not hex: %v", err)
	}
	if a == b {
		t.Fatal("two tokens must differ")
	}
}

func TestHashToken(t *testing.T) {
	h := hashToken("abc")

	if h != hashToken("abc") {
		t.Fatal("same token must give same hash")
	}
	if h == hashToken("abd") {
		t.Fatal("different tokens must give different hashes")
	}
	if h == "abc" || len(h) != 64 {
		t.Fatalf("want 64-char sha256 hex, got %q", h)
	}
	if h != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		t.Fatalf("unexpected sha256: %s", h)
	}
}
