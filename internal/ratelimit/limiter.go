package ratelimit

import (
	"sync"

	"golang.org/x/time/rate"
)

type Limiter struct {
	mu       sync.Mutex
	visitors map[string]*rate.Limiter
	r        rate.Limit
	b        int
}

func New(r rate.Limit, b int) *Limiter {
	return &Limiter{
		visitors: make(map[string]*rate.Limiter),
		r:        r,
		b:        b,
	}
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	lim, ok := l.visitors[key]
	if !ok {
		lim = rate.NewLimiter(l.r, l.b)
		l.visitors[key] = lim
	}

	return lim.Allow()
}
