// Package ratelimit is a small in-process fixed-window failure limiter for
// sensitive endpoints such as login: repeated failures for the same key are
// blocked until the window elapses. State is per-process memory — enough to
// blunt credential stuffing on a single node, not a distributed rate limiter.
package ratelimit

import (
	"sync"
	"time"
)

type window struct {
	fails   int
	resetAt time.Time
}

type Limiter struct {
	mu     sync.Mutex
	fails  map[string]*window
	max    int
	window time.Duration
	now    func() time.Time
}

// New returns a limiter that blocks a key after max failures within ttl.
func New(max int, ttl time.Duration) *Limiter {
	return &Limiter{
		fails:  map[string]*window{},
		max:    max,
		window: ttl,
		now:    time.Now,
	}
}

// Allowed reports whether an attempt for key may proceed. It never records a
// failure — call Fail after the attempt actually failed, Reset after it
// succeeded.
func (l *Limiter) Allowed(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	w, ok := l.fails[key]
	if !ok || !l.now().Before(w.resetAt) {
		return true
	}
	return w.fails < l.max
}

// Fail records a failed attempt for key, starting a fresh window when the
// previous one has already elapsed.
func (l *Limiter) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	w, ok := l.fails[key]
	if !ok || !now.Before(w.resetAt) {
		l.fails[key] = &window{fails: 1, resetAt: now.Add(l.window)}
		return
	}
	w.fails++
}

// Reset clears the failure counter for key, e.g. after a successful login.
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.fails, key)
}
