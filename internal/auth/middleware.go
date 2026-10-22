package auth

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const userIDKey ctxKey = "userID"

type Middleware struct {
	jwtSecret []byte
}

func NewMiddleware(jwtSecret []byte) *Middleware {
	return &Middleware{jwtSecret: jwtSecret}
}

func (m *Middleware) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok {
			http.Error(
				w,
				"missing token",
				http.StatusUnauthorized,
			)
			return
		}

		claims := &jwt.RegisteredClaims{}
		_, err := jwt.ParseWithClaims(
			token,
			claims,
			func(t *jwt.Token) (any, error) {
				return m.jwtSecret, nil
			},
			jwt.WithValidMethods([]string{"HS256"}),
		)
		if err != nil {
			http.Error(
				w,
				"invalid token",
				http.StatusUnauthorized,
			)
			return
		}

		id, err := strconv.Atoi(claims.Subject)
		if err != nil {
			http.Error(
				w,
				"invalid token",
				http.StatusUnauthorized,
			)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) (int, bool) {
	id, ok := ctx.Value(userIDKey).(int)
	return id, ok
}
