package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareAuth(t *testing.T) {
	validToken, err := createToken(7, testSecret)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	tests := []struct {
		name       string
		header     string
		wantStatus int
		wantUserID int
	}{
		{"ok", "Bearer " + validToken, http.StatusOK, 7},
		{"no header", "", http.StatusUnauthorized, 0},
		{"no bearer prefix", validToken, http.StatusUnauthorized, 0},
		{"bad token", "Bearer abc", http.StatusUnauthorized, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotID int
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				id, ok := UserIDFromContext(r.Context())
				if !ok {
					t.Fatal("user id not in context")
				}
				gotID = id
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()

			NewMiddleware(testSecret).Auth(next).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("want status %d, got %d", tt.wantStatus, rec.Code)
			}
			if gotID != tt.wantUserID {
				t.Fatalf("want user id %d, got %d", tt.wantUserID, gotID)
			}
		})
	}
}
