package store

import (
	"testing"
	"time"

	"github.com/yourusername/signalfence/core"
)

// TestRedisStore_BasicOperations tests Redis store operations
// Note: This requires a Redis instance running on localhost:6379
// Skip with: go test -short
func TestRedisStore_BasicOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test")
	}

	store := NewRedisStore(RedisConfig{
		Addr: "localhost:6379",
		DB:   15, // Use separate DB for tests
		TTL:  1 * time.Minute,
	})

	// Test connection
	if err := store.Ping(); err != nil {
		t.Skip("Redis not available:", err)
	}

	// Clean up before test
	store.Clear()
	defer store.Clear()

	// Test Set and Get
	state := &core.BucketState{
		Tokens:       10.5,
		LastRefillAt: time.Now(),
	}

	store.Set("test-key", state)
	retrieved := store.Get("test-key")

	if retrieved == nil {
		t.Fatal("Failed to retrieve state from Redis")
	}

	if retrieved.Tokens != state.Tokens {
		t.Errorf("Tokens = %.2f, want %.2f", retrieved.Tokens, state.Tokens)
	}

	// Test Delete
	store.Delete("test-key")
	if retrieved := store.Get("test-key"); retrieved != nil {
		t.Error("Key should be deleted")
	}

	// Test non-existent key
	if retrieved := store.Get("non-existent"); retrieved != nil {
		t.Error("Non-existent key should return nil")
	}
}

func TestRedisStore_MultipleKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test")
	}

	store := NewRedisStore(RedisConfig{
		Addr: "localhost:6379",
		DB:   15,
		TTL:  1 * time.Minute,
	})

	if err := store.Ping(); err != nil {
		t.Skip("Redis not available:", err)
	}

	store.Clear()
	defer store.Clear()

	// Store multiple keys
	keys := []string{"user1", "user2", "user3"}
	for i, key := range keys {
		state := &core.BucketState{
			Tokens:       float64(i + 1),
			LastRefillAt: time.Now(),
		}
		store.Set(key, state)
	}

	// Verify all keys
	for i, key := range keys {
		state := store.Get(key)
		if state == nil {
			t.Errorf("Key %s not found", key)
			continue
		}
		expected := float64(i + 1)
		if state.Tokens != expected {
			t.Errorf("Key %s: tokens = %.2f, want %.2f", key, state.Tokens, expected)
		}
	}
}
