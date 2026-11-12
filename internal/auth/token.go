package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	accessTTL  = 15 * time.Minute
	refreshTTL = 30 * 24 * time.Hour
)

func newRefreshToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf(
			"generate refresh token: %w",
			err,
		)
	}
	return hex.EncodeToString(buf), nil
}

func hashToken(token string) string {
	bytes := sha256.Sum256([]byte(token))

	return hex.EncodeToString(bytes[:])
}

func createToken(userID int, secret []byte) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   strconv.Itoa(userID),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTTL)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func ParseToken(token string, secret []byte) (int, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(
		token,
		claims,
		func(t *jwt.Token) (any, error) {
			return secret, nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
	)
	if err != nil {
		return 0, fmt.Errorf(
			"parse with claims: %w",
			err,
		)
	}

	id, err := strconv.Atoi(claims.Subject)
	if err != nil {
		return 0, fmt.Errorf(
			"strconv atoi: %w",
			err,
		)
	}
	return id, nil
}
