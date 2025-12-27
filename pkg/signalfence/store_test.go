package signalfence

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewInMemoryStore(t *testing.T) {
	tests := []struct {
		name        string
		config      BucketConfig
		cleanupAge  time.Duration
		wantErr     bool
		expectedErr error
	}{
		{
			name: "valid store",
			config: BucketConfig{
				Capacity:   100,
				RefillRate: 10.0,
			},
			cleanupAge: 1 * time.Hour,
			wantErr:    false,
		},
		{
			name: "zero cleanup age (cleanup disabled)",
			config: BucketConfig{
				Capacity:   100,
				RefillRate: 10.0,
			},
			cleanupAge: 0,
			wantErr:    false,
		},
		{
			name: "invalid capacity",
			config: BucketConfig{
				Capacity:   0,
				RefillRate: 10.0,
			},
			cleanupAge:  1 * time.Hour,
			wantErr:     true,
			expectedErr: ErrNegativeCapacity,
		},
		{
			name: "invalid refill rate",
			config: BucketConfig{
				Capacity:   100,
				RefillRate: 0,
			},
			cleanupAge:  1 * time.Hour,
			wantErr:     true,
			expectedErr: ErrNegativeRefillRate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewInMemoryStore(tt.config, tt.cleanupAge)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewInMemoryStore() expected error, got nil")
				}
				if err != tt.expectedErr {
					t.Errorf("NewInMemoryStore() error = %v, want %v", err, tt.expectedErr)
				}
				return
			}
			if err != nil {
				t.Errorf("NewInMemoryStore() unexpected error: %v", err)
				return
			}
			if store == nil {
				t.Fatal("NewInMemoryStore() returned nil store")
			}
			if store.Count() != 0 {
				t.Errorf("new store Count() = %d, want 0", store.Count())
			}
		})
	}
}

func TestInMemoryStore_GetBucket(t *testing.T) {
	config := BucketConfig{
		Capacity:   10,
		RefillRate: 1.0,
	}
	store, err := NewInMemoryStore(config, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewInMemoryStore() failed: %v", err)
	}

	// Get bucket for new key (should create it)
	bucket1, err := store.GetBucket("user1")
	if err != nil {
		t.Fatalf("GetBucket() failed: %v", err)
	}
	if bucket1 == nil {
		t.Fatal("GetBucket() returned nil bucket")
	}
	if bucket1.Capacity() != 10 {
		t.Errorf("bucket.Capacity() = %d, want 10", bucket1.Capacity())
	}

	// Store should now have 1 bucket
	if store.Count() != 1 {
		t.Errorf("store.Count() = %d, want 1", store.Count())
	}

	// Get same bucket again (should reuse existing)
	bucket2, err := store.GetBucket("user1")
	if err != nil {
		t.Fatalf("GetBucket() failed: %v", err)
	}
	if bucket1 != bucket2 {
		t.Error("GetBucket() should return same bucket for same key")
	}

	// Count should still be 1
	if store.Count() != 1 {
		t.Errorf("store.Count() = %d, want 1 (bucket reused)", store.Count())
	}

	// Get bucket for different key
	bucket3, err := store.GetBucket("user2")
	if err != nil {
		t.Fatalf("GetBucket() failed: %v", err)
	}
	if bucket1 == bucket3 {
		t.Error("GetBucket() should return different bucket for different key")
	}

	// Count should now be 2
	if store.Count() != 2 {
		t.Errorf("store.Count() = %d, want 2", store.Count())
	}
}

func TestInMemoryStore_GetBucket_EmptyKey(t *testing.T) {
	config := BucketConfig{
		Capacity:   10,
		RefillRate: 1.0,
	}
	store, err := NewInMemoryStore(config, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewInMemoryStore() failed: %v", err)
	}

	// Empty key should return error
	_, err = store.GetBucket("")
	if err != ErrInvalidKey {
		t.Errorf("GetBucket(\"\") error = %v, want %v", err, ErrInvalidKey)
	}
}

func TestInMemoryStore_Cleanup(t *testing.T) {
	config := BucketConfig{
		Capacity:   10,
		RefillRate: 1.0,
	}
	// Set cleanup age to 100ms for fast testing
	store, err := NewInMemoryStore(config, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("NewInMemoryStore() failed: %v", err)
	}

	// Create some buckets
	store.GetBucket("user1")
	store.GetBucket("user2")
	store.GetBucket("user3")

	if store.Count() != 3 {
		t.Fatalf("store.Count() = %d, want 3", store.Count())
	}

	// Immediately cleanup (should remove nothing)
	removed, err := store.Cleanup()
	if err != nil {
		t.Errorf("Cleanup() failed: %v", err)
	}
	if removed != 0 {
		t.Errorf("Cleanup() removed %d buckets, want 0 (not old enough)", removed)
	}

	// Wait for buckets to age
	time.Sleep(150 * time.Millisecond)

	// Cleanup should remove all idle buckets
	removed, err = store.Cleanup()
	if err != nil {
		t.Errorf("Cleanup() failed: %v", err)
	}
	if removed != 3 {
		t.Errorf("Cleanup() removed %d buckets, want 3", removed)
	}
	if store.Count() != 0 {
		t.Errorf("store.Count() = %d, want 0 after cleanup", store.Count())
	}
}

func TestInMemoryStore_Cleanup_RecentlyAccessed(t *testing.T) {
	config := BucketConfig{
		Capacity:   10,
		RefillRate: 1.0,
	}
	store, err := NewInMemoryStore(config, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("NewInMemoryStore() failed: %v", err)
	}

	// Create buckets
	store.GetBucket("user1")
	store.GetBucket("user2")

	// Wait for buckets to age
	time.Sleep(150 * time.Millisecond)

	// Access user1 (should update lastAccessed)
	store.GetBucket("user1")

	// Cleanup should only remove user2
	removed, err := store.Cleanup()
	if err != nil {
		t.Errorf("Cleanup() failed: %v", err)
	}
	if removed != 1 {
		t.Errorf("Cleanup() removed %d buckets, want 1 (only user2)", removed)
	}
	if store.Count() != 1 {
		t.Errorf("store.Count() = %d, want 1 (user1 kept)", store.Count())
	}

	// Verify user1 still exists
	bucket, _ := store.GetBucket("user1")
	if bucket == nil {
		t.Error("user1 bucket should still exist")
	}
}

func TestInMemoryStore_Cleanup_Disabled(t *testing.T) {
	config := BucketConfig{
		Capacity:   10,
		RefillRate: 1.0,
	}
	// Cleanup disabled with cleanupAge = 0
	store, err := NewInMemoryStore(config, 0)
	if err != nil {
		t.Fatalf("NewInMemoryStore() failed: %v", err)
	}

	store.GetBucket("user1")
	store.GetBucket("user2")

	// Cleanup should do nothing
	removed, err := store.Cleanup()
	if err != nil {
		t.Errorf("Cleanup() failed: %v", err)
	}
	if removed != 0 {
		t.Errorf("Cleanup() removed %d buckets, want 0 (cleanup disabled)", removed)
	}
	if store.Count() != 2 {
		t.Errorf("store.Count() = %d, want 2", store.Count())
	}
}

func TestInMemoryStore_Concurrent(t *testing.T) {
	config := BucketConfig{
		Capacity:   100,
		RefillRate: 10.0,
	}
	store, err := NewInMemoryStore(config, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewInMemoryStore() failed: %v", err)
	}

	// Launch many goroutines accessing buckets concurrently
	var wg sync.WaitGroup
	keys := 100
	goroutinesPerKey := 10

	for i := 0; i < keys; i++ {
		key := fmt.Sprintf("user%d", i)
		for j := 0; j < goroutinesPerKey; j++ {
			wg.Add(1)
			go func(k string) {
				defer wg.Done()
				bucket, err := store.GetBucket(k)
				if err != nil {
					t.Errorf("GetBucket(%s) failed: %v", k, err)
					return
				}
				if bucket == nil {
					t.Errorf("GetBucket(%s) returned nil", k)
					return
				}
				// Use the bucket
				bucket.Allow()
			}(key)
		}
	}

	wg.Wait()

	// Should have exactly 100 buckets (one per key)
	if store.Count() != keys {
		t.Errorf("store.Count() = %d, want %d", store.Count(), keys)
	}
}

func TestInMemoryStore_ConcurrentCleanup(t *testing.T) {
	config := BucketConfig{
		Capacity:   10,
		RefillRate: 1.0,
	}
	store, err := NewInMemoryStore(config, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewInMemoryStore() failed: %v", err)
	}

	// Create initial buckets
	for i := 0; i < 50; i++ {
		store.GetBucket(fmt.Sprintf("user%d", i))
	}

	var wg sync.WaitGroup

	// Concurrent GetBucket calls
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("user%d", idx%50)
			store.GetBucket(key)
		}(i)
	}

	// Concurrent Cleanup calls
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(60 * time.Millisecond)
			store.Cleanup()
		}()
	}

	wg.Wait()

	// Test passes if no race conditions (run with -race flag)
	count := store.Count()
	if count < 0 {
		t.Errorf("store.Count() = %d, should never be negative", count)
	}
}

func TestInMemoryStore_BackgroundCleanup(t *testing.T) {
	config := BucketConfig{
		Capacity:   10,
		RefillRate: 1.0,
	}
	store, err := NewInMemoryStore(config, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewInMemoryStore() failed: %v", err)
	}

	// Create buckets
	store.GetBucket("user1")
	store.GetBucket("user2")

	// Start background cleanup
	stop := store.StartBackgroundCleanup(100 * time.Millisecond)
	defer stop()

	// Wait for buckets to age and cleanup to run
	time.Sleep(200 * time.Millisecond)

	// Buckets should be cleaned up
	if store.Count() != 0 {
		t.Errorf("store.Count() = %d, want 0 (background cleanup should have run)", store.Count())
	}
}

func TestInMemoryStore_BackgroundCleanup_Disabled(t *testing.T) {
	config := BucketConfig{
		Capacity:   10,
		RefillRate: 1.0,
	}
	// Cleanup disabled
	store, err := NewInMemoryStore(config, 0)
	if err != nil {
		t.Fatalf("NewInMemoryStore() failed: %v", err)
	}

	// Start background cleanup (should be no-op)
	stop := store.StartBackgroundCleanup(100 * time.Millisecond)
	defer stop()

	store.GetBucket("user1")

	time.Sleep(200 * time.Millisecond)

	// Bucket should still exist (cleanup disabled)
	if store.Count() != 1 {
		t.Errorf("store.Count() = %d, want 1 (cleanup disabled)", store.Count())
	}
}

func TestInMemoryStore_BucketReuse(t *testing.T) {
	config := BucketConfig{
		Capacity:   10,
		RefillRate: 1.0,
	}
	store, err := NewInMemoryStore(config, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewInMemoryStore() failed: %v", err)
	}

	// Get bucket and consume some tokens
	bucket1, _ := store.GetBucket("user1")
	bucket1.AllowN(5)
	remaining1 := bucket1.Remaining()

	// Get same bucket again
	bucket2, _ := store.GetBucket("user1")
	remaining2 := bucket2.Remaining()

	// Should be same bucket (state preserved)
	if remaining1 != remaining2 {
		t.Errorf("bucket state not preserved: first remaining = %d, second = %d", remaining1, remaining2)
	}
	if bucket1 != bucket2 {
		t.Error("GetBucket should return same instance for same key")
	}
}
