package ratelimit

import (
	"net"
	"net/http"
	"strconv"

	"github.com/cdlinkin/pingpong/internal/auth"
)

func (l *Limiter) ByIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}

		if !l.Allow(host) {
			w.Header().Set("Retry-After", "1")
			http.Error(
				w,
				"too many requests",
				http.StatusTooManyRequests,
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (l *Limiter) ByUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok {
			http.Error(
				w,
				"unauthorized",
				http.StatusUnauthorized,
			)
			return
		}

		userIDStr := strconv.Itoa(userID)

		if !l.Allow(userIDStr) {
			w.Header().Set("Retry-After", "1")
			http.Error(
				w,
				"too many requests",
				http.StatusTooManyRequests,
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}
