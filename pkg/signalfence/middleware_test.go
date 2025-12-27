package signalfence

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestMiddleware_AllowedRequest(t *testing.T) {
	limiter, err := NewRateLimiter(
		WithDefaults(5, 1.0),
	)
	if err != nil {
		t.Fatalf("NewRateLimiter() failed: %v", err)
	}

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should be allowed
	if rr.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	// Check headers
	if rr.Header().Get("X-RateLimit-Limit") != "5" {
		t.Errorf("X-RateLimit-Limit = %s, want 5", rr.Header().Get("X-RateLimit-Limit"))
	}
	if rr.Header().Get("X-RateLimit-Remaining") != "4" {
		t.Errorf("X-RateLimit-Remaining = %s, want 4", rr.Header().Get("X-RateLimit-Remaining"))
	}

	// Body should be "success"
	if rr.Body.String() != "success" {
		t.Errorf("body = %s, want success", rr.Body.String())
	}
}

func TestMiddleware_RateLimited(t *testing.T) {
	limiter, err := NewRateLimiter(
		WithDefaults(3, 0.1), // 3 tokens, slow refill
	)
	if err != nil {
		t.Fatalf("NewRateLimiter() failed: %v", err)
	}

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	// First 3 requests should succeed
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("request %d: status code = %d, want %d", i+1, rr.Code, http.StatusOK)
		}
	}

	// 4th request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("status code = %d, want %d", rr.Code, http.StatusTooManyRequests)
	}

	// Check rate limit headers
	if rr.Header().Get("X-RateLimit-Limit") != "3" {
		t.Errorf("X-RateLimit-Limit = %s, want 3", rr.Header().Get("X-RateLimit-Limit"))
	}
	if rr.Header().Get("X-RateLimit-Remaining") != "0" {
		t.Errorf("X-RateLimit-Remaining = %s, want 0", rr.Header().Get("X-RateLimit-Remaining"))
	}

	// Should have Retry-After header
	retryAfter := rr.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Error("Retry-After header should be set when rate limited")
	}

	// Should have X-RateLimit-Reset header
	resetHeader := rr.Header().Get("X-RateLimit-Reset")
	if resetHeader == "" {
		t.Error("X-RateLimit-Reset header should be set when rate limited")
	}

	// Verify reset time is in the future
	resetTime, err := strconv.ParseInt(resetHeader, 10, 64)
	if err != nil {
		t.Errorf("X-RateLimit-Reset parsing failed: %v", err)
	}
	if resetTime <= time.Now().Unix() {
		t.Error("X-RateLimit-Reset should be in the future")
	}
}

func TestMiddleware_DifferentIPs(t *testing.T) {
	limiter, err := NewRateLimiter(
		WithDefaults(2, 1.0),
	)
	if err != nil {
		t.Fatalf("NewRateLimiter() failed: %v", err)
	}

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust tokens for IP1
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	// IP1 should be rate limited
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusTooManyRequests {
		t.Errorf("IP1 should be rate limited, got status %d", rr1.Code)
	}

	// IP2 should still be allowed (separate bucket)
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.2:12345"
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("IP2 should be allowed, got status %d", rr2.Code)
	}
}

func TestMiddleware_WithAPIKey(t *testing.T) {
	limiter, err := NewRateLimiter(
		WithDefaults(3, 1.0),
		WithKeyExtractor(ExtractHeader("X-API-Key")),
	)
	if err != nil {
		t.Fatalf("NewRateLimiter() failed: %v", err)
	}

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Requests with API key should be tracked separately
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-API-Key", "key123")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("request %d should be allowed, got status %d", i+1, rr.Code)
		}
	}

	// 4th request with same API key should be limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "key123")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("4th request should be rate limited, got status %d", rr.Code)
	}

	// Different API key should have separate bucket
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("X-API-Key", "key456")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("different API key should be allowed, got status %d", rr2.Code)
	}
}

func TestMiddleware_MissingAPIKey(t *testing.T) {
	limiter, err := NewRateLimiter(
		WithDefaults(3, 1.0),
		WithKeyExtractor(ExtractHeader("X-API-Key")),
	)
	if err != nil {
		t.Fatalf("NewRateLimiter() failed: %v", err)
	}

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request without API key should return 500 (key extraction failed)
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status code = %d, want %d (missing API key)", rr.Code, http.StatusInternalServerError)
	}
}

func TestMiddleware_HeadersAlwaysSet(t *testing.T) {
	limiter, err := NewRateLimiter(
		WithDefaults(10, 1.0),
	)
	if err != nil {
		t.Fatalf("NewRateLimiter() failed: %v", err)
	}

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Headers should be set even for allowed requests
	if rr.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("X-RateLimit-Limit header should always be set")
	}
	if rr.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("X-RateLimit-Remaining header should always be set")
	}

	// Retry-After should NOT be set for allowed requests
	if rr.Header().Get("Retry-After") != "" {
		t.Error("Retry-After should not be set for allowed requests")
	}
}

func TestMiddleware_MultipleRoutes(t *testing.T) {
	limiter, err := NewRateLimiter(
		WithDefaults(5, 1.0),
	)
	if err != nil {
		t.Fatalf("NewRateLimiter() failed: %v", err)
	}

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make requests to different routes from same IP
	routes := []string{"/api/users", "/api/posts", "/api/comments"}

	for _, route := range routes {
		req := httptest.NewRequest("GET", route, nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("request to %s should be allowed, got status %d", route, rr.Code)
		}
	}

	// All routes share the same bucket (same IP)
	// Remaining should be 5 - 3 = 2
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("X-RateLimit-Remaining") != "1" {
		t.Errorf("X-RateLimit-Remaining = %s, want 1", rr.Header().Get("X-RateLimit-Remaining"))
	}
}

func TestMiddleware_Concurrent(t *testing.T) {
	limiter, err := NewRateLimiter(
		WithDefaults(100, 10.0),
	)
	if err != nil {
		t.Fatalf("NewRateLimiter() failed: %v", err)
	}

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make concurrent requests
	successCount := 0
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code == http.StatusOK {
			successCount++
		}
	}

	// Should allow exactly 100 requests (capacity)
	if successCount != 100 {
		t.Errorf("allowed %d requests, want 100", successCount)
	}
}
