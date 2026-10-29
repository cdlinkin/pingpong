package chats

import (
	"context"
	"errors"
	"testing"
)

type memberKey struct{ chatID, userID int }

type fakeRepo struct {
	nextID  int
	roles   map[memberKey]string
	direct  map[[2]int]int
	created int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		roles:  make(map[memberKey]string),
		direct: make(map[[2]int]int),
	}
}

func pair(a, b int) [2]int {
	if a > b {
		a, b = b, a
	}
	return [2]int{a, b}
}

func (f *fakeRepo) CreateGroupChat(ctx context.Context, title string, creatorID int) (int, error) {
	f.nextID++
	f.roles[memberKey{f.nextID, creatorID}] = "admin"
	return f.nextID, nil
}

func (f *fakeRepo) CreateDirectChat(ctx context.Context, userA, userB int) (int, error) {
	f.nextID++
	f.created++
	f.direct[pair(userA, userB)] = f.nextID
	f.roles[memberKey{f.nextID, userA}] = "member"
	f.roles[memberKey{f.nextID, userB}] = "member"
	return f.nextID, nil
}

func (f *fakeRepo) FindDirectChat(ctx context.Context, userA, userB int) (int, error) {
	id, ok := f.direct[pair(userA, userB)]
	if !ok {
		return 0, ErrChatNotFound
	}
	return id, nil
}

func (f *fakeRepo) ListUserChats(ctx context.Context, userID int) ([]ChatModel, error) {
	var chats []ChatModel
	for k := range f.roles {
		if k.userID == userID {
			chats = append(chats, ChatModel{ID: k.chatID})
		}
	}
	return chats, nil
}

func (f *fakeRepo) GetMemberRole(ctx context.Context, chatID, userID int) (string, error) {
	role, ok := f.roles[memberKey{chatID, userID}]
	if !ok {
		return "", ErrNotMember
	}
	return role, nil
}

func (f *fakeRepo) AddMember(ctx context.Context, chatID, userID int) error {
	if _, ok := f.roles[memberKey{chatID, userID}]; ok {
		return ErrAlreadyMember
	}
	f.roles[memberKey{chatID, userID}] = "member"
	return nil
}

func (f *fakeRepo) RemoveMember(ctx context.Context, chatID, userID int) error {
	if _, ok := f.roles[memberKey{chatID, userID}]; !ok {
		return ErrNotMember
	}
	delete(f.roles, memberKey{chatID, userID})
	return nil
}

func TestCreateGroupChat(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	svc := NewService(repo)

	id, err := svc.CreateGroupChat(ctx, "  Go team ", 1)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if repo.roles[memberKey{id, 1}] != "admin" {
		t.Fatal("creator must be admin")
	}

	if _, err := svc.CreateGroupChat(ctx, "   ", 1); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestCreateDirectChat(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	svc := NewService(repo)

	t.Run("with yourself", func(t *testing.T) {
		_, _, err := svc.CreateDirectChat(ctx, 1, 1)
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("want ErrInvalidInput, got %v", err)
		}
	})

	firstID, created, err := svc.CreateDirectChat(ctx, 1, 2)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	if !created {
		t.Fatal("first call must create chat")
	}

	t.Run("second call returns same chat", func(t *testing.T) {
		id, created, err := svc.CreateDirectChat(ctx, 2, 1)
		if err != nil {
			t.Fatalf("second create: %v", err)
		}
		if created {
			t.Fatal("second call must not create chat")
		}
		if id != firstID {
			t.Fatalf("want chat %d, got %d", firstID, id)
		}
		if repo.created != 1 {
			t.Fatalf("want 1 chat in repo, got %d", repo.created)
		}
	})
}

func TestAddMember(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	svc := NewService(repo)

	const admin, member, stranger, newUser = 1, 2, 3, 4
	chatID, _ := svc.CreateGroupChat(ctx, "Go team", admin)
	repo.roles[memberKey{chatID, member}] = "member"

	tests := []struct {
		name      string
		requester int
		chatID    int
		target    int
		wantErr   error
	}{
		{"admin adds", admin, chatID, newUser, nil},
		{"member is forbidden", member, chatID, 5, ErrForbidden},
		{"stranger sees not found", stranger, chatID, 5, ErrChatNotFound},
		{"chat does not exist", admin, 999, 5, ErrChatNotFound},
		{"already member", admin, chatID, member, ErrAlreadyMember},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.AddMember(ctx, tt.requester, tt.chatID, tt.target)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("want %v, got %v", tt.wantErr, err)
			}
		})
	}

	if _, ok := repo.roles[memberKey{chatID, newUser}]; !ok {
		t.Fatal("new user was not added")
	}
	if _, ok := repo.roles[memberKey{chatID, 5}]; ok {
		t.Fatal("user 5 must not be added")
	}
}

func TestRemoveMember(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	svc := NewService(repo)

	const admin, member, other, stranger = 1, 2, 3, 4
	chatID, _ := svc.CreateGroupChat(ctx, "Go team", admin)
	repo.roles[memberKey{chatID, member}] = "member"
	repo.roles[memberKey{chatID, other}] = "member"

	tests := []struct {
		name      string
		requester int
		target    int
		wantErr   error
	}{
		{"remove yourself", admin, admin, ErrInvalidInput},
		{"member is forbidden", member, other, ErrForbidden},
		{"stranger sees not found", stranger, member, ErrChatNotFound},
		{"admin removes", admin, member, nil},
		{"target not in chat", admin, stranger, ErrNotMember},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.RemoveMember(ctx, tt.requester, chatID, tt.target)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("want %v, got %v", tt.wantErr, err)
			}
		})
	}

	if _, ok := repo.roles[memberKey{chatID, member}]; ok {
		t.Fatal("member was not removed")
	}
	if _, ok := repo.roles[memberKey{chatID, other}]; !ok {
		t.Fatal("other must stay in chat")
	}
}
