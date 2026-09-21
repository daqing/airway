package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterAllowsWithinMax(t *testing.T) {
	l := New(3, time.Minute)

	for i := 0; i < 3; i++ {
		if !l.Allowed("a@x") {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
		l.Fail("a@x")
	}

	if l.Allowed("a@x") {
		t.Fatal("attempt after max failures should be blocked")
	}
	if !l.Allowed("b@x") {
		t.Fatal("a different key should not be blocked")
	}
}

func TestLimiterResetOnSuccess(t *testing.T) {
	l := New(2, time.Minute)

	l.Fail("a@x")
	l.Fail("a@x")
	if l.Allowed("a@x") {
		t.Fatal("should be blocked after two failures")
	}

	l.Reset("a@x")
	if !l.Allowed("a@x") {
		t.Fatal("should be allowed after reset")
	}
}

func TestLimiterWindowExpiry(t *testing.T) {
	l := New(1, time.Minute)
	now := time.Now()
	l.now = func() time.Time { return now }

	l.Fail("a@x")
	if l.Allowed("a@x") {
		t.Fatal("should be blocked inside the window")
	}

	now = now.Add(2 * time.Minute)
	if !l.Allowed("a@x") {
		t.Fatal("should be allowed after the window elapses")
	}

	// A new failure inside the fresh window starts counting from one again.
	l.Fail("a@x")
	l.Fail("a@x")
	if l.Allowed("a@x") {
		t.Fatal("should be blocked again after two fresh failures")
	}
}
