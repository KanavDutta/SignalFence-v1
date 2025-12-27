package signalfence

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractIP(t *testing.T) {
	extractor := ExtractIP()

	tests := []struct {
		name       string
		remoteAddr string
		want       string
		wantErr    bool
	}{
		{
			name:       "valid IP with port",
			remoteAddr: "192.168.1.1:12345",
			want:       "ip:192.168.1.1",
			wantErr:    false,
		},
		{
			name:       "valid IP without port",
			remoteAddr: "192.168.1.1",
			want:       "ip:192.168.1.1",
			wantErr:    false,
		},
		{
			name:       "IPv6 with port",
			remoteAddr: "[2001:db8::1]:8080",
			want:       "ip:2001:db8::1",
			wantErr:    false,
		},
		{
			name:       "localhost",
			remoteAddr: "127.0.0.1:54321",
			want:       "ip:127.0.0.1",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr

			got, err := extractor(req)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestExtractIPWithProxy(t *testing.T) {
	extractor := ExtractIPWithProxy()

	tests := []struct {
		name           string
		remoteAddr     string
		xForwardedFor  string
		xRealIP        string
		want           string
		wantErr        bool
	}{
		{
			name:           "X-Forwarded-For single IP",
			remoteAddr:     "192.168.1.1:12345",
			xForwardedFor:  "203.0.113.1",
			want:           "ip:203.0.113.1",
			wantErr:        false,
		},
		{
			name:           "X-Forwarded-For multiple IPs",
			remoteAddr:     "192.168.1.1:12345",
			xForwardedFor:  "203.0.113.1, 192.168.1.254, 10.0.0.1",
			want:           "ip:203.0.113.1",
			wantErr:        false,
		},
		{
			name:           "X-Forwarded-For with spaces",
			remoteAddr:     "192.168.1.1:12345",
			xForwardedFor:  "  203.0.113.1  ",
			want:           "ip:203.0.113.1",
			wantErr:        false,
		},
		{
			name:           "X-Real-IP (no X-Forwarded-For)",
			remoteAddr:     "192.168.1.1:12345",
			xRealIP:        "203.0.113.2",
			want:           "ip:203.0.113.2",
			wantErr:        false,
		},
		{
			name:           "X-Forwarded-For takes precedence over X-Real-IP",
			remoteAddr:     "192.168.1.1:12345",
			xForwardedFor:  "203.0.113.1",
			xRealIP:        "203.0.113.2",
			want:           "ip:203.0.113.1",
			wantErr:        false,
		},
		{
			name:           "fallback to RemoteAddr",
			remoteAddr:     "192.168.1.1:12345",
			want:           "ip:192.168.1.1",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			got, err := extractor(req)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestExtractHeader(t *testing.T) {
	tests := []struct {
		name        string
		headerName  string
		headerValue string
		want        string
		wantErr     bool
	}{
		{
			name:        "X-API-Key present",
			headerName:  "X-API-Key",
			headerValue: "abc123",
			want:        "header:X-API-Key:abc123",
			wantErr:     false,
		},
		{
			name:        "custom header",
			headerName:  "X-Client-ID",
			headerValue: "client-456",
			want:        "header:X-Client-ID:client-456",
			wantErr:     false,
		},
		{
			name:       "header missing",
			headerName: "X-API-Key",
			wantErr:    true,
		},
		{
			name:        "header empty",
			headerName:  "X-API-Key",
			headerValue: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := ExtractHeader(tt.headerName)
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.headerValue != "" {
				req.Header.Set(tt.headerName, tt.headerValue)
			}

			got, err := extractor(req)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestExtractBearer(t *testing.T) {
	extractor := ExtractBearer()

	tests := []struct {
		name        string
		authHeader  string
		want        string
		wantErr     bool
	}{
		{
			name:       "valid bearer token",
			authHeader: "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			want:       "bearer:eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantErr:    false,
		},
		{
			name:       "bearer with lowercase",
			authHeader: "bearer mytoken123",
			want:       "bearer:mytoken123",
			wantErr:    false,
		},
		{
			name:       "missing Authorization header",
			authHeader: "",
			wantErr:    true,
		},
		{
			name:       "invalid format (no space)",
			authHeader: "Bearertoken123",
			wantErr:    true,
		},
		{
			name:       "wrong auth type",
			authHeader: "Basic dXNlcjpwYXNz",
			wantErr:    true,
		},
		{
			name:       "empty token",
			authHeader: "Bearer ",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			got, err := extractor(req)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestExtractComposite(t *testing.T) {
	t.Run("first extractor succeeds", func(t *testing.T) {
		extractor := ExtractComposite(
			ExtractHeader("X-API-Key"),
			ExtractIP(),
		)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-API-Key", "key123")
		req.RemoteAddr = "192.168.1.1:12345"

		got, err := extractor(req)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !strings.HasPrefix(got, "header:X-API-Key") {
			t.Errorf("expected header key, got %s", got)
		}
	})

	t.Run("fallback to second extractor", func(t *testing.T) {
		extractor := ExtractComposite(
			ExtractHeader("X-API-Key"),
			ExtractIP(),
		)

		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		// No X-API-Key header, should fall back to IP

		got, err := extractor(req)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !strings.HasPrefix(got, "ip:") {
			t.Errorf("expected IP key, got %s", got)
		}
	})

	t.Run("all extractors fail", func(t *testing.T) {
		extractor := ExtractComposite(
			ExtractHeader("X-API-Key"),
			ExtractHeader("X-Client-ID"),
		)

		req := httptest.NewRequest("GET", "/test", nil)
		// No headers set

		_, err := extractor(req)
		if err == nil {
			t.Error("expected error when all extractors fail, got nil")
		}
	})

	t.Run("no extractors provided", func(t *testing.T) {
		extractor := ExtractComposite()
		req := httptest.NewRequest("GET", "/test", nil)

		_, err := extractor(req)
		if err == nil {
			t.Error("expected error for no extractors, got nil")
		}
	})
}

func TestExtractStatic(t *testing.T) {
	tests := []struct {
		name      string
		staticKey string
		wantErr   bool
	}{
		{
			name:      "valid static key",
			staticKey: "global",
			wantErr:   false,
		},
		{
			name:      "empty static key",
			staticKey: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := ExtractStatic(tt.staticKey)
			req := httptest.NewRequest("GET", "/test", nil)

			got, err := extractor(req)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if got != tt.staticKey {
				t.Errorf("got %s, want %s", got, tt.staticKey)
			}
		})
	}
}

func TestExtractCookie(t *testing.T) {
	tests := []struct {
		name        string
		cookieName  string
		cookieValue string
		want        string
		wantErr     bool
	}{
		{
			name:        "valid cookie",
			cookieName:  "session_id",
			cookieValue: "sess123",
			want:        "cookie:session_id:sess123",
			wantErr:     false,
		},
		{
			name:       "cookie not found",
			cookieName: "session_id",
			wantErr:    true,
		},
		{
			name:        "empty cookie value",
			cookieName:  "session_id",
			cookieValue: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := ExtractCookie(tt.cookieName)
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.cookieValue != "" {
				req.AddCookie(&http.Cookie{
					Name:  tt.cookieName,
					Value: tt.cookieValue,
				})
			}

			got, err := extractor(req)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestParseKeyExtractorConfig(t *testing.T) {
	tests := []struct {
		name       string
		config     string
		wantErr    bool
		testValue  string // Expected prefix or value to test
	}{
		{
			name:      "ip extractor",
			config:    "ip",
			wantErr:   false,
			testValue: "ip:",
		},
		{
			name:      "ip-proxy extractor",
			config:    "ip-proxy",
			wantErr:   false,
			testValue: "ip:",
		},
		{
			name:      "header extractor",
			config:    "header:X-API-Key",
			wantErr:   false,
			testValue: "header:X-API-Key:",
		},
		{
			name:      "bearer extractor",
			config:    "bearer",
			wantErr:   false,
			testValue: "bearer:",
		},
		{
			name:      "cookie extractor",
			config:    "cookie:session_id",
			wantErr:   false,
			testValue: "cookie:session_id:",
		},
		{
			name:      "static extractor",
			config:    "static:global",
			wantErr:   false,
			testValue: "global",
		},
		{
			name:    "unknown extractor type",
			config:  "unknown",
			wantErr: true,
		},
		{
			name:    "header without name",
			config:  "header",
			wantErr: true,
		},
		{
			name:    "cookie without name",
			config:  "cookie",
			wantErr: true,
		},
		{
			name:    "static without key",
			config:  "static",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor, err := ParseKeyExtractorConfig(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if extractor == nil {
				t.Fatal("extractor is nil")
			}

			// Test the extractor with a mock request
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			req.Header.Set("X-API-Key", "testkey")
			req.Header.Set("Authorization", "Bearer testtoken")
			req.AddCookie(&http.Cookie{Name: "session_id", Value: "testsession"})

			got, err := extractor(req)
			if err != nil && tt.testValue != "" {
				t.Errorf("extractor failed: %v", err)
				return
			}
			if tt.testValue != "" && !strings.Contains(got, tt.testValue) {
				t.Errorf("key %s doesn't contain expected value %s", got, tt.testValue)
			}
		})
	}
}
