package session_api

import (
	"sync"
	"time"
)

// attempt stores the state of a rate limit key within its current time window.
type attempt struct {
	count   int
	resetAt time.Time
}

// RateLimiter provides concurrency-safe in-process fixed-window limiting for login requests.
type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string]attempt
	limit    int
	window   time.Duration
	now      func() time.Time
}

// NewRateLimiter creates a limiter with the specified attempt limit and time window.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{attempts: map[string]attempt{}, limit: limit, window: window, now: time.Now}
}

// Allow records an attempt and reports whether the rate limit key may proceed.
func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now()
	current := r.attempts[key]
	if current.resetAt.IsZero() || !now.Before(current.resetAt) {
		current = attempt{resetAt: now.Add(r.window)}
	}
	if current.count >= r.limit {
		r.attempts[key] = current
		return false
	}
	current.count++
	r.attempts[key] = current
	return true
}

// Reset clears the attempt history for a rate limit key after a successful login.
func (r *RateLimiter) Reset(key string) { r.mu.Lock(); delete(r.attempts, key); r.mu.Unlock() }
