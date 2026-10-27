package auth

import (
	"context"
	"net/http"
	"strings"
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

		id, err := ParseToken(token, m.jwtSecret)
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
