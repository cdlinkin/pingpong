package ws

import (
	"sync"
	"testing"
)

func newClient(userID int) *Client {
	return &Client{UserID: userID, Send: make(chan []byte, 1)}
}

func TestHubSendToUser(t *testing.T) {
	hub := NewHub()
	tab1 := newClient(1)
	tab2 := newClient(1)
	other := newClient(2)
	hub.Register(tab1)
	hub.Register(tab2)
	hub.Register(other)

	hub.SendToUser(1, []byte("hi"))

	for i, c := range []*Client{tab1, tab2} {
		select {
		case got := <-c.Send:
			if string(got) != "hi" {
				t.Fatalf("tab %d: want hi, got %s", i+1, got)
			}
		default:
			t.Fatalf("tab %d did not receive message", i+1)
		}
	}

	select {
	case got := <-other.Send:
		t.Fatalf("other user must not receive message, got %s", got)
	default:
	}
}

func TestHubSendDoesNotBlockOnFullBuffer(t *testing.T) {
	hub := NewHub()
	c := newClient(1)
	hub.Register(c)

	hub.SendToUser(1, []byte("first"))
	hub.SendToUser(1, []byte("second"))

	if got := <-c.Send; string(got) != "first" {
		t.Fatalf("want first, got %s", got)
	}
}

func TestHubUnregister(t *testing.T) {
	hub := NewHub()
	c := newClient(1)
	hub.Register(c)

	hub.Unregister(c)

	if _, ok := <-c.Send; ok {
		t.Fatal("Send channel must be closed")
	}
	if _, ok := hub.pool[1]; ok {
		t.Fatal("empty user entry must be removed from pool")
	}

	hub.Unregister(c)

	hub.SendToUser(1, []byte("hi"))
}

func TestHubConcurrent(t *testing.T) {
	hub := NewHub()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			c := newClient(userID % 5)
			hub.Register(c)
			hub.SendToUser(userID%5, []byte("x"))
			hub.Unregister(c)
		}(i)
	}
	wg.Wait()

	if len(hub.pool) != 0 {
		t.Fatalf("pool must be empty, got %d users", len(hub.pool))
	}
}
