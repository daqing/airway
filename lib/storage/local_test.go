package storage

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestLocalRoundTrip(t *testing.T) {
	store, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}

	ctx := context.Background()
	key := "docs/202607/hello.txt"

	if err := store.Put(ctx, key, strings.NewReader("hello storage"), int64(len("hello storage")), "text/plain"); err != nil {
		t.Fatalf("Put: %v", err)
	}

	exists, err := store.Exists(ctx, key)
	if err != nil || !exists {
		t.Fatalf("Exists after Put: %v, %v", exists, err)
	}

	r, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer r.Close()

	content, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if string(content) != "hello storage" {
		t.Fatalf("unexpected content: %q", content)
	}

	url, err := store.URL(ctx, key, 0)
	if err != nil {
		t.Fatalf("URL: %v", err)
	}

	if url != "/api/v1/storage/docs/202607/hello.txt" {
		t.Fatalf("unexpected URL: %q", url)
	}

	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	exists, err = store.Exists(ctx, key)
	if err != nil || exists {
		t.Fatalf("Exists after Delete: %v, %v", exists, err)
	}
}

func TestLocalDeleteMissingKeyIsNoop(t *testing.T) {
	store, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}

	if err := store.Delete(context.Background(), "missing.txt"); err != nil {
		t.Fatalf("Delete missing key should be a no-op, got: %v", err)
	}
}

func TestLocalRejectsInvalidKeys(t *testing.T) {
	store, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}

	ctx := context.Background()

	for _, key := range []string{"", "../escape.txt", "..", "/abs.txt", "a/../../escape.txt"} {
		if err := store.Put(ctx, key, strings.NewReader("x"), 1, ""); err == nil {
			t.Fatalf("Put should reject key %q", key)
		}

		if _, err := store.Get(ctx, key); err == nil {
			t.Fatalf("Get should reject key %q", key)
		}
	}
}
