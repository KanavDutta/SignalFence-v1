package signalfence

import (
	"sync"
	"testing"
	"time"
)

func TestNewBucket(t *testing.T) {
	tests := []struct {
		name        string
		capacity    int64
		refillRate  float64
		wantErr     bool
		expectedErr error
	}{
		{
			name:       "valid bucket",
			capacity:   100,
			refillRate: 10.0,
			wantErr:    false,
		},
		{
			name:        "zero capacity",
			capacity:    0,
			refillRate:  10.0,
			wantErr:     true,
			expectedErr: ErrNegativeCapacity,
		},
		{
			name:        "negative capacity",
			capacity:    -10,
			refillRate:  10.0,
			wantErr:     true,
			expectedErr: ErrNegativeCapacity,
		},
		{
			name:        "zero refill rate",
			capacity:    100,
			refillRate:  0,
			wantErr:     true,
			expectedErr: ErrNegativeRefillRate,
		},
		{
			name:        "negative refill rate",
			capacity:    100,
			refillRate:  -5.0,
			wantErr:     true,
			expectedErr: ErrNegativeRefillRate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bucket, err := NewBucket(tt.capacity, tt.refillRate)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewBucket() expected error, got nil")
				}
				if err != tt.expectedErr {
					t.Errorf("NewBucket() error = %v, want %v", err, tt.expectedErr)
				}
				return
			}
			if err != nil {
				t.Errorf("NewBucket() unexpected error: %v", err)
				return
			}
			if bucket == nil {
				t.Fatal("NewBucket() returned nil bucket")
			}
			if bucket.Capacity() != tt.capacity {
				t.Errorf("bucket.Capacity() = %d, want %d", bucket.Capacity(), tt.capacity)
			}
			if bucket.RefillRate() != tt.refillRate {
				t.Errorf("bucket.RefillRate() = %f, want %f", bucket.RefillRate(), tt.refillRate)
			}
			// Bucket should start full
			if bucket.Remaining() != tt.capacity {
				t.Errorf("bucket.Remaining() = %d, want %d (full)", bucket.Remaining(), tt.capacity)
			}
		})
	}
}

func TestBucket_Allow(t *testing.T) {
	bucket, err := NewBucket(3, 1.0)
	if err != nil {
		t.Fatalf("NewBucket() failed: %v", err)
	}

	// First 3 requests should succeed
	for i := 0; i < 3; i++ {
		if !bucket.Allow() {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 4th request should be denied (bucket empty)
	if bucket.Allow() {
		t.Error("4th request should be denied (bucket empty)")
	}

	// Remaining should be 0
	if remaining := bucket.Remaining(); remaining != 0 {
		t.Errorf("bucket.Remaining() = %d, want 0", remaining)
	}
}

func TestBucket_AllowN(t *testing.T) {
	bucket, err := NewBucket(10, 1.0)
	if err != nil {
		t.Fatalf("NewBucket() failed: %v", err)
	}

	// Consume 3 tokens
	if !bucket.AllowN(3) {
		t.Error("AllowN(3) should succeed")
	}
	if remaining := bucket.Remaining(); remaining != 7 {
		t.Errorf("bucket.Remaining() = %d, want 7", remaining)
	}

	// Consume 7 more tokens
	if !bucket.AllowN(7) {
		t.Error("AllowN(7) should succeed")
	}
	if remaining := bucket.Remaining(); remaining != 0 {
		t.Errorf("bucket.Remaining() = %d, want 0", remaining)
	}

	// Try to consume 1 more token (should fail)
	if bucket.AllowN(1) {
		t.Error("AllowN(1) should fail (bucket empty)")
	}
}

func TestBucket_Refill(t *testing.T) {
	// Create bucket with 10 tokens, refilling at 10 tokens/second
	bucket, err := NewBucket(10, 10.0)
	if err != nil {
		t.Fatalf("NewBucket() failed: %v", err)
	}

	// Consume all tokens
	for i := 0; i < 10; i++ {
		if !bucket.Allow() {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// Bucket should be empty
	if bucket.Allow() {
		t.Error("bucket should be empty")
	}

	// Wait 100ms (should refill 1 token at 10 tokens/sec)
	time.Sleep(100 * time.Millisecond)

	// Should allow 1 request now
	if !bucket.Allow() {
		t.Error("request should be allowed after refill")
	}

	// Should deny next request (only 1 token refilled)
	if bucket.Allow() {
		t.Error("request should be denied (only 1 token refilled)")
	}

	// Wait 1 second (should refill to capacity)
	time.Sleep(1 * time.Second)

	// Should allow 10 requests (full capacity)
	for i := 0; i < 10; i++ {
		if !bucket.Allow() {
			t.Errorf("request %d should be allowed after full refill", i+1)
		}
	}
}

func TestBucket_BurstBehavior(t *testing.T) {
	// Create bucket with burst capacity of 100, but slow refill (1/sec)
	bucket, err := NewBucket(100, 1.0)
	if err != nil {
		t.Fatalf("NewBucket() failed: %v", err)
	}

	// Should allow 100 requests immediately (burst)
	for i := 0; i < 100; i++ {
		if !bucket.Allow() {
			t.Errorf("burst request %d should be allowed", i+1)
		}
	}

	// 101st request should fail
	if bucket.Allow() {
		t.Error("request exceeding burst should be denied")
	}
}

func TestBucket_RefillCap(t *testing.T) {
	// Create bucket with small capacity
	bucket, err := NewBucket(5, 10.0)
	if err != nil {
		t.Fatalf("NewBucket() failed: %v", err)
	}

	// Wait long enough to theoretically add 100 tokens
	time.Sleep(10 * time.Second)

	// Should only have capacity (5 tokens), not unlimited
	successCount := 0
	for i := 0; i < 100; i++ {
		if bucket.Allow() {
			successCount++
		}
	}

	if successCount != 5 {
		t.Errorf("allowed %d requests, want 5 (capped at capacity)", successCount)
	}
}

func TestBucket_Concurrent(t *testing.T) {
	// Create bucket with high capacity, very low refill rate to minimize refill during test
	bucket, err := NewBucket(1000, 0.1)
	if err != nil {
		t.Fatalf("NewBucket() failed: %v", err)
	}

	// Launch 100 goroutines, each making 10 requests
	var wg sync.WaitGroup
	allowedCount := 0
	var mu sync.Mutex

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				if bucket.Allow() {
					mu.Lock()
					allowedCount++
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	// Should allow at least 1000 requests (capacity), possibly a few more due to refill
	if allowedCount < 1000 {
		t.Errorf("allowed %d requests, want at least 1000", allowedCount)
	}

	// Allow a small margin for refill during concurrent execution (at 0.1 tokens/sec, very minimal)
	if allowedCount > 1010 {
		t.Errorf("allowed %d requests, want at most ~1010 (accounting for minimal refill)", allowedCount)
	}

	// Bucket should be nearly empty (remaining should be very small)
	remaining := bucket.Remaining()
	if remaining > 5 {
		t.Errorf("bucket.Remaining() = %d, should be nearly empty (< 5)", remaining)
	}
}

func TestBucket_ConcurrentStress(t *testing.T) {
	// Stress test with many concurrent goroutines
	bucket, err := NewBucket(10000, 1000.0)
	if err != nil {
		t.Fatalf("NewBucket() failed: %v", err)
	}

	var wg sync.WaitGroup
	goroutines := 500
	requestsPerGoroutine := 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				bucket.Allow()
			}
		}()
	}

	wg.Wait()

	// Test passes if no race conditions detected (run with -race flag)
	// Remaining should be non-negative
	remaining := bucket.Remaining()
	if remaining < 0 {
		t.Errorf("bucket.Remaining() = %d, should never be negative", remaining)
	}
}

func TestBucket_RetryAfter(t *testing.T) {
	// Create bucket with 1 token, refilling at 10 tokens/second
	bucket, err := NewBucket(1, 10.0)
	if err != nil {
		t.Fatalf("NewBucket() failed: %v", err)
	}

	// When bucket has tokens, RetryAfter should be 0
	if retry := bucket.RetryAfter(); retry != 0 {
		t.Errorf("RetryAfter() = %v, want 0 (bucket has tokens)", retry)
	}

	// Consume the token
	bucket.Allow()

	// Now RetryAfter should be ~100ms (1 token / 10 tokens/sec)
	retry := bucket.RetryAfter()
	expectedRetry := 100 * time.Millisecond
	tolerance := 10 * time.Millisecond

	if retry < expectedRetry-tolerance || retry > expectedRetry+tolerance {
		t.Errorf("RetryAfter() = %v, want ~%v", retry, expectedRetry)
	}
}

func TestBucket_Remaining(t *testing.T) {
	bucket, err := NewBucket(10, 1.0)
	if err != nil {
		t.Fatalf("NewBucket() failed: %v", err)
	}

	// Initially full
	if remaining := bucket.Remaining(); remaining != 10 {
		t.Errorf("Remaining() = %d, want 10", remaining)
	}

	// After consuming 3
	bucket.AllowN(3)
	if remaining := bucket.Remaining(); remaining != 7 {
		t.Errorf("Remaining() = %d, want 7", remaining)
	}

	// After consuming all
	bucket.AllowN(7)
	if remaining := bucket.Remaining(); remaining != 0 {
		t.Errorf("Remaining() = %d, want 0", remaining)
	}
}

func TestBucket_FractionalRefill(t *testing.T) {
	// Test fractional refill rates (e.g., 0.5 tokens/second = 30 req/min)
	bucket, err := NewBucket(10, 0.5)
	if err != nil {
		t.Fatalf("NewBucket() failed: %v", err)
	}

	// Consume all tokens
	bucket.AllowN(10)

	// Wait 2 seconds (should refill 1 token at 0.5 tokens/sec)
	time.Sleep(2 * time.Second)

	if !bucket.Allow() {
		t.Error("should allow 1 request after 2 seconds (0.5 tokens/sec)")
	}

	if bucket.Allow() {
		t.Error("should deny next request (only 1 token refilled)")
	}
}
