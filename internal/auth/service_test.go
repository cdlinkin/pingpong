package auth

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type fakeRefresh struct {
	userID    int
	expiresAt time.Time
}

type fakeRepo struct {
	users     map[string]UserModel
	createErr error
	refresh   map[string]fakeRefresh
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{users: make(map[string]UserModel), refresh: make(map[string]fakeRefresh)}
}

func (f *fakeRepo) SaveRefreshToken(ctx context.Context, userID int, hash string, expiresAt time.Time) error {
	f.refresh[hash] = fakeRefresh{userID: userID, expiresAt: expiresAt}
	return nil
}

func (f *fakeRepo) ConsumeRefreshToken(ctx context.Context, hash string) (int, error) {
	rt, ok := f.refresh[hash]
	if !ok || !rt.expiresAt.After(time.Now()) {
		return 0, ErrInvalidRefreshToken
	}
	delete(f.refresh, hash)
	return rt.userID, nil
}

func (f *fakeRepo) DeleteRefreshToken(ctx context.Context, hash string) error {
	delete(f.refresh, hash)
	return nil
}

func (f *fakeRepo) CreateUser(ctx context.Context, login, email, passwordHash string) (int, error) {
	if f.createErr != nil {
		return 0, f.createErr
	}
	if _, ok := f.users[email]; ok {
		return 0, ErrUserExists
	}
	id := len(f.users) + 1
	f.users[email] = UserModel{ID: id, Login: login, Email: email, PasswordHash: passwordHash}
	return id, nil
}

func (f *fakeRepo) GetUserByEmail(ctx context.Context, email string) (UserModel, error) {
	u, ok := f.users[email]
	if !ok {
		return UserModel{}, ErrUserNotFound
	}
	return u, nil
}

func (f *fakeRepo) GetUserByID(ctx context.Context, id int) (UserModel, error) {
	for _, u := range f.users {
		if u.ID == id {
			return u, nil
		}
	}
	return UserModel{}, ErrUserNotFound
}

func TestRegister(t *testing.T) {
	ctx := context.Background()

	t.Run("ok, password is hashed, email normalized", func(t *testing.T) {
		repo := newFakeRepo()
		svc := NewService(repo, testSecret)

		id, err := svc.Register(ctx, "kuroame", " Kuro@Mail.ru ", "password123")
		if err != nil {
			t.Fatalf("register: %v", err)
		}
		if id != 1 {
			t.Fatalf("want id 1, got %d", id)
		}

		u, ok := repo.users["kuro@mail.ru"]
		if !ok {
			t.Fatal("user not saved with normalized email")
		}
		if u.PasswordHash == "password123" {
			t.Fatal("password saved in plain text")
		}
		if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte("password123")); err != nil {
			t.Fatalf("hash does not match password: %v", err)
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		svc := NewService(newFakeRepo(), testSecret)
		if _, err := svc.Register(ctx, "kuroame", "kuro@mail.ru", "password123"); err != nil {
			t.Fatalf("first register: %v", err)
		}
		_, err := svc.Register(ctx, "other", "kuro@mail.ru", "password123")
		if !errors.Is(err, ErrUserExists) {
			t.Fatalf("want ErrUserExists, got %v", err)
		}
	})

	invalid := []struct {
		name, login, email, password string
	}{
		{"bad email", "kuroame", "bad", "password123"},
		{"bad login", "ab", "kuro@mail.ru", "password123"},
		{"bad password", "kuroame", "kuro@mail.ru", "123"},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			_, err := NewService(repo, testSecret).Register(ctx, tt.login, tt.email, tt.password)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
			if len(repo.users) != 0 {
				t.Fatal("repository must not be called on invalid input")
			}
		})
	}
}

func TestLogin(t *testing.T) {
	ctx := context.Background()
	svc := NewService(newFakeRepo(), testSecret)
	userID, err := svc.Register(ctx, "kuroame", "kuro@mail.ru", "password123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	t.Run("ok returns token with user id", func(t *testing.T) {
		tokens, err := svc.Login(ctx, "KURO@mail.ru", "password123")
		if err != nil {
			t.Fatalf("login: %v", err)
		}
		if tokens.Refresh == "" {
			t.Fatal("want refresh token")
		}
		id, err := ParseToken(tokens.Access, testSecret)
		if err != nil {
			t.Fatalf("parse token: %v", err)
		}
		if id != userID {
			t.Fatalf("want id %d, got %d", userID, id)
		}
	})

	tests := []struct {
		name, email, password string
		wantErr               error
	}{
		{"wrong password", "kuro@mail.ru", "wrongpass1", ErrInvalidCredentials},
		{"unknown email", "nobody@mail.ru", "password123", ErrInvalidCredentials},
		{"invalid email", "bad", "password123", ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Login(ctx, tt.email, tt.password)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("want %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestMe(t *testing.T) {
	ctx := context.Background()
	svc := NewService(newFakeRepo(), testSecret)
	id, _ := svc.Register(ctx, "kuroame", "kuro@mail.ru", "password123")

	u, err := svc.Me(ctx, id)
	if err != nil {
		t.Fatalf("me: %v", err)
	}
	if u.Login != "kuroame" {
		t.Fatalf("want login kuroame, got %q", u.Login)
	}

	if _, err := svc.Me(ctx, 999); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("want ErrUserNotFound, got %v", err)
	}
}

func TestRefresh(t *testing.T) {
	ctx := context.Background()

	login := func(t *testing.T) (*fakeRepo, *Service, int, TokenPair) {
		t.Helper()
		repo := newFakeRepo()
		svc := NewService(repo, testSecret)
		userID, err := svc.Register(ctx, "kuroame", "kuro@mail.ru", "password123")
		if err != nil {
			t.Fatalf("register: %v", err)
		}
		tokens, err := svc.Login(ctx, "kuro@mail.ru", "password123")
		if err != nil {
			t.Fatalf("login: %v", err)
		}
		return repo, svc, userID, tokens
	}

	t.Run("refresh is stored as hash", func(t *testing.T) {
		repo, _, _, tokens := login(t)
		if _, ok := repo.refresh[tokens.Refresh]; ok {
			t.Fatal("raw refresh token must not be stored")
		}
		if _, ok := repo.refresh[hashToken(tokens.Refresh)]; !ok {
			t.Fatal("hash of refresh token must be stored")
		}
	})

	t.Run("ok gives new pair and rotates refresh", func(t *testing.T) {
		_, svc, userID, tokens := login(t)
		next, err := svc.Refresh(ctx, tokens.Refresh)
		if err != nil {
			t.Fatalf("refresh: %v", err)
		}
		if next.Refresh == tokens.Refresh {
			t.Fatal("refresh token must rotate")
		}
		id, err := ParseToken(next.Access, testSecret)
		if err != nil || id != userID {
			t.Fatalf("want access for user %d, got %d (%v)", userID, id, err)
		}
		if _, err := svc.Refresh(ctx, tokens.Refresh); !errors.Is(err, ErrInvalidRefreshToken) {
			t.Fatalf("old refresh must be dead, got %v", err)
		}
		if _, err := svc.Refresh(ctx, next.Refresh); err != nil {
			t.Fatalf("new refresh must work: %v", err)
		}
	})

	t.Run("expired refresh", func(t *testing.T) {
		repo, svc, _, tokens := login(t)
		h := hashToken(tokens.Refresh)
		rt := repo.refresh[h]
		rt.expiresAt = time.Now().Add(-time.Minute)
		repo.refresh[h] = rt
		if _, err := svc.Refresh(ctx, tokens.Refresh); !errors.Is(err, ErrInvalidRefreshToken) {
			t.Fatalf("want ErrInvalidRefreshToken, got %v", err)
		}
	})

	for _, bad := range []string{"", "not-a-token"} {
		t.Run("bad token "+strconv.Quote(bad), func(t *testing.T) {
			_, svc, _, _ := login(t)
			if _, err := svc.Refresh(ctx, bad); !errors.Is(err, ErrInvalidRefreshToken) {
				t.Fatalf("want ErrInvalidRefreshToken, got %v", err)
			}
		})
	}
}

func TestLogout(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	svc := NewService(repo, testSecret)
	if _, err := svc.Register(ctx, "kuroame", "kuro@mail.ru", "password123"); err != nil {
		t.Fatalf("register: %v", err)
	}
	tokens, err := svc.Login(ctx, "kuro@mail.ru", "password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if err := svc.Logout(ctx, tokens.Refresh); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := svc.Refresh(ctx, tokens.Refresh); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("refresh after logout: want ErrInvalidRefreshToken, got %v", err)
	}
	if err := svc.Logout(ctx, tokens.Refresh); err != nil {
		t.Fatalf("second logout must be ok: %v", err)
	}
	if err := svc.Logout(ctx, ""); err != nil {
		t.Fatalf("empty logout must be ok: %v", err)
	}
}
