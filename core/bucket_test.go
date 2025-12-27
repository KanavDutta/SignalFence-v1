package core

import (
	"testing"
	"time"
)

func TestTokenBucket_AllowsBurstRequests(t *testing.T) {
	config := Config{
		Capacity:     10,
		RefillPerSec: 5,
	}
	bucket := NewTokenBucket(config)
	now := time.Now()

	var state *BucketState

	// Should allow up to capacity requests instantly
	for i := 0; i < 10; i++ {
		var result CheckResult
		state, result = bucket.Check(state, now)
		
		if !result.Allowed {
			t.Errorf("Request %d should be allowed (burst)", i+1)
		}
	}

	// 11th request should be blocked
	_, result := bucket.Check(state, now)
	if result.Allowed {
		t.Error("Request 11 should be blocked (bucket empty)")
	}
}

func TestTokenBucket_BlocksWhenEmpty(t *testing.T) {
	config := Config{
		Capacity:     5,
		RefillPerSec: 2,
	}
	bucket := NewTokenBucket(config)
	now := time.Now()

	var state *BucketState

	// Drain the bucket
	for i := 0; i < 5; i++ {
		state, _ = bucket.Check(state, now)
	}

	// Next request should be blocked
	state, result := bucket.Check(state, now)
	if result.Allowed {
		t.Error("Request should be blocked when bucket is empty")
	}
	if result.RetryAfterMs <= 0 {
		t.Error("RetryAfterMs should be positive when blocked")
	}
}

func TestTokenBucket_RefillsOverTime(t *testing.T) {
	config := Config{
		Capacity:     10,
		RefillPerSec: 5, // 5 tokens per second = 1 token per 200ms
	}
	bucket := NewTokenBucket(config)
	now := time.Now()

	var state *BucketState

	// Drain the bucket
	for i := 0; i < 10; i++ {
		state, _ = bucket.Check(state, now)
	}

	// Should be blocked immediately
	state, result := bucket.Check(state, now)
	if result.Allowed {
		t.Error("Should be blocked immediately after draining")
	}

	// Wait 1 second (should refill 5 tokens)
	now = now.Add(1 * time.Second)
	
	// Should allow 5 requests
	for i := 0; i < 5; i++ {
		var result CheckResult
		state, result = bucket.Check(state, now)
		if !result.Allowed {
			t.Errorf("Request %d should be allowed after refill", i+1)
		}
	}

	// 6th request should be blocked
	_, result = bucket.Check(state, now)
	if result.Allowed {
		t.Error("Request should be blocked after using refilled tokens")
	}
}

func TestTokenBucket_CorrectRetryAfter(t *testing.T) {
	config := Config{
		Capacity:     5,
		RefillPerSec: 2, // 1 token every 500ms
	}
	bucket := NewTokenBucket(config)
	now := time.Now()

	var state *BucketState

	// Drain the bucket
	for i := 0; i < 5; i++ {
		state, _ = bucket.Check(state, now)
	}

	// Check retry time
	_, result := bucket.Check(state, now)
	
	// Should need ~500ms to get 1 token (at 2 tokens/sec)
	expectedMs := int64(500)
	tolerance := int64(50) // Allow some tolerance
	
	if result.RetryAfterMs < expectedMs-tolerance || result.RetryAfterMs > expectedMs+tolerance {
		t.Errorf("RetryAfterMs = %d, want ~%d", result.RetryAfterMs, expectedMs)
	}
}

func TestTokenBucket_CapsAtCapacity(t *testing.T) {
	config := Config{
		Capacity:     10,
		RefillPerSec: 5,
	}
	bucket := NewTokenBucket(config)
	now := time.Now()

	// Start with empty bucket
	state := &BucketState{
		Tokens:       0,
		LastRefillAt: now,
	}

	// Wait 10 seconds (would refill 50 tokens, but capped at 10)
	now = now.Add(10 * time.Second)
	state, result := bucket.Check(state, now)

	// Should have capacity tokens, not more
	if !result.Allowed {
		t.Error("Request should be allowed after long wait")
	}
	
	// Remaining should be capacity - 1 (we just consumed 1)
	expected := config.Capacity - 1
	if result.Remaining != expected {
		t.Errorf("Remaining = %.2f, want %.2f", result.Remaining, expected)
	}
}
