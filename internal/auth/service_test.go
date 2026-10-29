package auth

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

type fakeRepo struct {
	users     map[string]UserModel
	createErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{users: make(map[string]UserModel)}
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
		token, err := svc.Login(ctx, "KURO@mail.ru", "password123")
		if err != nil {
			t.Fatalf("login: %v", err)
		}
		id, err := ParseToken(token, testSecret)
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
