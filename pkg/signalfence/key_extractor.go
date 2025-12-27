package signalfence

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

// KeyExtractor is a function that extracts a rate limit key from an HTTP request.
// The key is used to identify the client (e.g., IP address, API key, user ID).
type KeyExtractor func(*http.Request) (string, error)

// ExtractIP returns a KeyExtractor that uses the client's IP address.
// It uses r.RemoteAddr which includes the port.
func ExtractIP() KeyExtractor {
	return func(r *http.Request) (string, error) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			// RemoteAddr might not have a port in some edge cases
			ip = r.RemoteAddr
		}
		if ip == "" {
			return "", fmt.Errorf("%w: empty IP address", ErrKeyExtractionFailed)
		}
		return "ip:" + ip, nil
	}
}

// ExtractIPWithProxy returns a KeyExtractor that considers proxy headers.
// It checks X-Forwarded-For and X-Real-IP headers before falling back to RemoteAddr.
// This is important when the application is behind a reverse proxy or load balancer.
func ExtractIPWithProxy() KeyExtractor {
	return func(r *http.Request) (string, error) {
		// Check X-Forwarded-For header (most common)
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// X-Forwarded-For can be a comma-separated list of IPs
			// The first IP is the original client IP
			ips := strings.Split(xff, ",")
			if len(ips) > 0 {
				ip := strings.TrimSpace(ips[0])
				if ip != "" {
					return "ip:" + ip, nil
				}
			}
		}

		// Check X-Real-IP header (alternative)
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return "ip:" + xri, nil
		}

		// Fallback to RemoteAddr
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		if ip == "" {
			return "", fmt.Errorf("%w: empty IP address", ErrKeyExtractionFailed)
		}
		return "ip:" + ip, nil
	}
}

// ExtractHeader returns a KeyExtractor that uses a specific HTTP header.
// Example: ExtractHeader("X-API-Key") will use the X-API-Key header value.
func ExtractHeader(headerName string) KeyExtractor {
	return func(r *http.Request) (string, error) {
		value := r.Header.Get(headerName)
		if value == "" {
			return "", fmt.Errorf("%w: header %s not found or empty", ErrKeyExtractionFailed, headerName)
		}
		return fmt.Sprintf("header:%s:%s", headerName, value), nil
	}
}

// ExtractBearer returns a KeyExtractor that uses the Bearer token from Authorization header.
// Expects header format: "Authorization: Bearer <token>"
func ExtractBearer() KeyExtractor {
	return func(r *http.Request) (string, error) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			return "", fmt.Errorf("%w: Authorization header not found", ErrKeyExtractionFailed)
		}

		// Expected format: "Bearer <token>"
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return "", fmt.Errorf("%w: invalid Authorization header format", ErrKeyExtractionFailed)
		}

		token := parts[1]
		if token == "" {
			return "", fmt.Errorf("%w: empty bearer token", ErrKeyExtractionFailed)
		}

		return "bearer:" + token, nil
	}
}

// ExtractComposite returns a KeyExtractor that tries multiple extractors in order.
// It returns the key from the first extractor that succeeds.
// This is useful for fallback behavior (e.g., try API key, then fall back to IP).
//
// Example:
//
//	extractor := ExtractComposite(
//	    ExtractHeader("X-API-Key"),
//	    ExtractIPWithProxy(),  // Fallback to IP if no API key
//	)
func ExtractComposite(extractors ...KeyExtractor) KeyExtractor {
	if len(extractors) == 0 {
		return func(r *http.Request) (string, error) {
			return "", fmt.Errorf("%w: no extractors provided", ErrKeyExtractionFailed)
		}
	}

	return func(r *http.Request) (string, error) {
		var lastErr error
		for _, extractor := range extractors {
			key, err := extractor(r)
			if err == nil && key != "" {
				return key, nil
			}
			lastErr = err
		}
		if lastErr != nil {
			return "", fmt.Errorf("%w: all extractors failed: %v", ErrKeyExtractionFailed, lastErr)
		}
		return "", fmt.Errorf("%w: all extractors returned empty key", ErrKeyExtractionFailed)
	}
}

// ExtractStatic returns a KeyExtractor that always returns the same key.
// This is useful for global rate limiting (all clients share the same limit).
//
// Example:
//
//	extractor := ExtractStatic("global")
func ExtractStatic(key string) KeyExtractor {
	return func(r *http.Request) (string, error) {
		if key == "" {
			return "", fmt.Errorf("%w: static key is empty", ErrKeyExtractionFailed)
		}
		return key, nil
	}
}

// ExtractCookie returns a KeyExtractor that uses a specific cookie value.
// Example: ExtractCookie("session_id")
func ExtractCookie(cookieName string) KeyExtractor {
	return func(r *http.Request) (string, error) {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			return "", fmt.Errorf("%w: cookie %s not found: %v", ErrKeyExtractionFailed, cookieName, err)
		}
		if cookie.Value == "" {
			return "", fmt.Errorf("%w: cookie %s has empty value", ErrKeyExtractionFailed, cookieName)
		}
		return fmt.Sprintf("cookie:%s:%s", cookieName, cookie.Value), nil
	}
}

// ParseKeyExtractorConfig creates a KeyExtractor from a configuration string.
// Supported formats:
// - "ip" -> ExtractIP()
// - "ip-proxy" -> ExtractIPWithProxy()
// - "header:X-API-Key" -> ExtractHeader("X-API-Key")
// - "bearer" -> ExtractBearer()
// - "cookie:session_id" -> ExtractCookie("session_id")
// - "static:global" -> ExtractStatic("global")
func ParseKeyExtractorConfig(config string) (KeyExtractor, error) {
	parts := strings.SplitN(config, ":", 2)

	switch parts[0] {
	case "ip":
		return ExtractIP(), nil

	case "ip-proxy":
		return ExtractIPWithProxy(), nil

	case "header":
		if len(parts) != 2 {
			return nil, fmt.Errorf("%w: header extractor requires format 'header:HeaderName'", ErrInvalidConfig)
		}
		return ExtractHeader(parts[1]), nil

	case "bearer":
		return ExtractBearer(), nil

	case "cookie":
		if len(parts) != 2 {
			return nil, fmt.Errorf("%w: cookie extractor requires format 'cookie:CookieName'", ErrInvalidConfig)
		}
		return ExtractCookie(parts[1]), nil

	case "static":
		if len(parts) != 2 {
			return nil, fmt.Errorf("%w: static extractor requires format 'static:key'", ErrInvalidConfig)
		}
		return ExtractStatic(parts[1]), nil

	default:
		return nil, fmt.Errorf("%w: unknown key extractor type: %s", ErrInvalidConfig, parts[0])
	}
}
