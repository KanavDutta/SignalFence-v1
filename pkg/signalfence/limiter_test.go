package signalfence

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "default rate limiter",
			opts:    nil,
			wantErr: false,
		},
		{
			name: "with defaults option",
			opts: []Option{
				WithDefaults(100, 10.0),
			},
			wantErr: false,
		},
		{
			name: "with config option",
			opts: []Option{
				WithConfig(NewConfig()),
			},
			wantErr: false,
		},
		{
			name: "with key extractor",
			opts: []Option{
				WithKeyExtractor(ExtractIPWithProxy()),
			},
			wantErr: false,
		},
		{
			name: "multiple options",
			opts: []Option{
				WithDefaults(50, 5.0),
				WithKeyExtractor(ExtractIP()),
				WithCleanupAge(30 * time.Minute),
			},
			wantErr: false,
		},
		{
			name: "invalid defaults (zero capacity)",
			opts: []Option{
				WithDefaults(0, 10.0),
			},
			wantErr: true,
		},
		{
			name: "invalid defaults (zero refill rate)",
			opts: []Option{
				WithDefaults(100, 0),
			},
			wantErr: true,
		},
		{
			name: "nil config",
			opts: []Option{
				WithConfig(nil),
			},
			wantErr: true,
		},
		{
			name: "nil key extractor",
			opts: []Option{
				WithKeyExtractor(nil),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter, err := NewRateLimiter(tt.opts...)
			if tt.wantErr {
				if err == nil {
					t.Error("NewRateLimiter() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("NewRateLimiter() unexpected error: %v", err)
				return
			}
			if limiter == nil {
				t.Fatal("NewRateLimiter() returned nil limiter")
			}
		})
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	limiter, err := NewRateLimiter(
		WithDefaults(3, 1.0),
	)
	if err != nil {
		t.Fatalf("NewRateLimiter() failed: %v", err)
	}

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		decision, err := limiter.Allow("testkey")
		if err != nil {
			t.Errorf("Allow() unexpected error: %v", err)
		}
		if !decision.Allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
		if decision.Limit != 3 {
			t.Errorf("decision.Limit = %d, want 3", decision.Limit)
		}
		if decision.Remaining != int64(2-i) {
			t.Errorf("decision.Remaining = %d, want %d", decision.Remaining, 2-i)
		}
		if decision.Key != "testkey" {
			t.Errorf("decision.Key = %s, want testkey", decision.Key)
		}
	}

	// 4th request should be denied
	decision, err := limiter.Allow("testkey")
	if err != nil {
		t.Fatalf("Allow() unexpected error: %v", err)
	}
	if decision.Allowed {
		t.Error("4th request should be denied")
	}
	if decision.Remaining != 0 {
		t.Errorf("decision.Remaining = %d, want 0", decision.Remaining)
	}
	if decision.RetryAfter == 0 {
		t.Error("decision.RetryAfter should be > 0 when denied")
	}
}

func TestRateLimiter_Allow_EmptyKey(t *testing.T) {
	limiter, err := NewRateLimiter()
	if err != nil {
		t.Fatalf("NewRateLimiter() failed: %v", err)
	}

	_, err = limiter.Allow("")
	if err != ErrInvalidKey {
		t.Errorf("Allow(\"\") error = %v, want %v", err, ErrInvalidKey)
	}
}

func TestRateLimiter_Allow_DifferentKeys(t *testing.T) {
	limiter, err := NewRateLimiter(
		WithDefaults(2, 1.0),
	)
	if err != nil {
		t.Fatalf("NewRateLimiter() failed: %v", err)
	}

	// Use all tokens for key1
	limiter.Allow("key1")
	limiter.Allow("key1")

	// key1 should be exhausted
	decision, _ := limiter.Allow("key1")
	if decision.Allowed {
		t.Error("key1 should be exhausted")
	}

	// key2 should still have tokens
	decision, _ = limiter.Allow("key2")
	if !decision.Allowed {
		t.Error("key2 should have tokens (separate bucket)")
	}
}

func TestRateLimiter_AllowRequest(t *testing.T) {
	limiter, err := NewRateLimiter(
		WithDefaults(5, 1.0),
		WithKeyExtractor(ExtractIP()),
	)
	if err != nil {
		t.Fatalf("NewRateLimiter() failed: %v", err)
	}

	// Create test request
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	// First request should be allowed
	decision, err := limiter.AllowRequest(req)
	if err != nil {
		t.Fatalf("AllowRequest() unexpected error: %v", err)
	}
	if !decision.Allowed {
		t.Error("first request should be allowed")
	}
	if decision.Route != "/api/test" {
		t.Errorf("decision.Route = %s, want /api/test", decision.Route)
	}
	if decision.Key == "" {
		t.Error("decision.Key should not be empty")
	}
}

func TestRateLimiter_AllowRequest_DisabledPolicy(t *testing.T) {
	config := NewConfig()
	config.Policies["/api/unlimited"] = PolicyConfig{
		Capacity:   100,
		RefillRate: 10.0,
		Enabled:    false, // Disabled
	}

	limiter, err := NewRateLimiter(
		WithConfig(config),
		WithKeyExtractor(ExtractIP()),
	)
	if err != nil {
		t.Fatalf("NewRateLimiter() failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/unlimited", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	// Should always allow (rate limiting disabled for this route)
	for i := 0; i < 200; i++ {
		decision, err := limiter.AllowRequest(req)
		if err != nil {
			t.Fatalf("AllowRequest() unexpected error: %v", err)
		}
		if !decision.Allowed {
			t.Errorf("request %d should be allowed (rate limiting disabled)", i+1)
		}
	}
}

func TestRateLimiter_AllowRequest_KeyExtractionFailed(t *testing.T) {
	limiter, err := NewRateLimiter(
		WithKeyExtractor(ExtractHeader("X-API-Key")),
	)
	if err != nil {
		t.Fatalf("NewRateLimiter() failed: %v", err)
	}

	// Request without X-API-Key header
	req := httptest.NewRequest("GET", "/api/test", nil)

	_, err = limiter.AllowRequest(req)
	if err == nil {
		t.Error("AllowRequest() expected error for missing header, got nil")
	}
}

func TestRateLimiter_AllowRequest_CompositeExtractor(t *testing.T) {
	limiter, err := NewRateLimiter(
		WithDefaults(2, 1.0),
		WithKeyExtractor(ExtractComposite(
			ExtractHeader("X-API-Key"),
			ExtractIP(), // Fallback
		)),
	)
	if err != nil {
		t.Fatalf("NewRateLimiter() failed: %v", err)
	}

	// Request with API key
	req1 := httptest.NewRequest("GET", "/api/test", nil)
	req1.Header.Set("X-API-Key", "key123")
	req1.RemoteAddr = "192.168.1.1:12345"

	decision1, err := limiter.AllowRequest(req1)
	if err != nil {
		t.Fatalf("AllowRequest() unexpected error: %v", err)
	}
	if !decision1.Allowed {
		t.Error("request with API key should be allowed")
	}

	// Request from same IP but different API key (should use separate bucket)
	req2 := httptest.NewRequest("GET", "/api/test", nil)
	req2.Header.Set("X-API-Key", "key456")
	req2.RemoteAddr = "192.168.1.1:12345"

	decision2, err := limiter.AllowRequest(req2)
	if err != nil {
		t.Fatalf("AllowRequest() unexpected error: %v", err)
	}
	if !decision2.Allowed {
		t.Error("request with different API key should be allowed (separate bucket)")
	}

	// Request without API key (should fall back to IP)
	req3 := httptest.NewRequest("GET", "/api/test", nil)
	req3.RemoteAddr = "192.168.1.2:12345"

	decision3, err := limiter.AllowRequest(req3)
	if err != nil {
		t.Fatalf("AllowRequest() unexpected error: %v", err)
	}
	if !decision3.Allowed {
		t.Error("request without API key should be allowed (IP fallback)")
	}
}

func TestRateLimiter_StartBackgroundCleanup(t *testing.T) {
	config := BucketConfig{
		Capacity:   10,
		RefillRate: 1.0,
	}
	store, err := NewInMemoryStore(config, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewInMemoryStore() failed: %v", err)
	}

	limiter, err := NewRateLimiter(
		WithStore(store),
		WithCleanupInterval(100 * time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewRateLimiter() failed: %v", err)
	}

	// Add some buckets
	limiter.Allow("key1")
	limiter.Allow("key2")

	if store.Count() != 2 {
		t.Fatalf("expected 2 buckets, got %d", store.Count())
	}

	// Start background cleanup
	stop := limiter.StartBackgroundCleanup()
	defer stop()

	// Wait for buckets to age and cleanup to run
	time.Sleep(200 * time.Millisecond)

	// Buckets should be cleaned up
	if store.Count() != 0 {
		t.Errorf("expected 0 buckets after cleanup, got %d", store.Count())
	}
}

func TestWithOptions(t *testing.T) {
	t.Run("WithStore", func(t *testing.T) {
		config := BucketConfig{Capacity: 10, RefillRate: 1.0}
		store, _ := NewInMemoryStore(config, 1*time.Hour)

		limiter, err := NewRateLimiter(WithStore(store))
		if err != nil {
			t.Errorf("WithStore() unexpected error: %v", err)
		}
		if limiter == nil {
			t.Error("limiter should not be nil")
		}
	})

	t.Run("WithConfigFile", func(t *testing.T) {
		// This requires a valid config file, tested in integration tests
		_, err := NewRateLimiter(WithConfigFile("nonexistent.yaml"))
		if err == nil {
			t.Error("WithConfigFile() expected error for nonexistent file, got nil")
		}
	})

	t.Run("WithCleanupInterval", func(t *testing.T) {
		limiter, err := NewRateLimiter(
			WithCleanupInterval(5 * time.Minute),
		)
		if err != nil {
			t.Errorf("WithCleanupInterval() unexpected error: %v", err)
		}
		if limiter == nil {
			t.Error("limiter should not be nil")
		}
	})

	t.Run("WithCleanupInterval negative", func(t *testing.T) {
		_, err := NewRateLimiter(
			WithCleanupInterval(-1 * time.Minute),
		)
		if err == nil {
			t.Error("WithCleanupInterval() expected error for negative interval, got nil")
		}
	})
}
