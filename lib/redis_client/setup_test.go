package redis_client

import "testing"

func TestSetupCreatesClientFromURL(t *testing.T) {
	Setup("redis://127.0.0.1:6379/0")

	rdb := RDB()
	if rdb == nil {
		t.Fatalf("expected redis client, got nil")
	}

	if addr := rdb.Options().Addr; addr != "127.0.0.1:6379" {
		t.Fatalf("expected addr %q, got %q", "127.0.0.1:6379", addr)
	}
}
