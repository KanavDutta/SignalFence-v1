# SignalFence 🚦

A lightweight, production-ready rate limiter for Go using the token bucket algorithm.

## What is SignalFence?

SignalFence is a "bouncer" for your backend API. Before your API does real work, it checks: "Has this user/IP/API key made too many requests too quickly?"

- ✅ **If no** → request goes through
- ❌ **If yes** → request is blocked with `429 Too Many Requests`

This protects your app from:
- Spam and bots
- Brute-force login attempts
- Accidental overload (broken clients)
- Unfair resource usage

## How It Works (Token Bucket Algorithm)

Each client has a "bucket" of tokens:
1. Bucket starts full (e.g., 20 tokens)
2. Each request costs 1 token
3. Tokens refill gradually over time (e.g., 1 token/sec)
4. If bucket is empty → block and tell them when to retry

This is industry-standard, practical, and efficient.

## Installation

```bash
go get github.com/yourusername/signalfence
```

## Quick Start

```go
package main

import (
    "net/http"
    "github.com/yourusername/signalfence"
)

func main() {
    // Create rate limiter: 10 requests per second, burst of 20
    limiter := signalfence.NewRateLimiter(signalfence.Config{
        Capacity:     20,
        RefillPerSec: 10,
    })

    // Wrap your handler
    http.Handle("/api/data", limiter.Middleware(http.HandlerFunc(dataHandler)))
    
    http.ListenAndServe(":8080", nil)
}

func dataHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte(`{"message": "success"}`))
}
```

## Testing

```bash
# Run tests
go test ./...

# With coverage
go test -cover ./...

# Run example server
go run examples/server.go
```

## Example Usage

```bash
# Normal request (should succeed)
curl http://localhost:8080/api/data

# Spam requests to trigger rate limit
for i in {1..25}; do curl http://localhost:8080/api/data; done
```

## Response Headers

SignalFence adds standard rate limit headers:

```
X-RateLimit-Limit: 20
X-RateLimit-Remaining: 15
Retry-After: 2
```

## Configuration

```go
config := signalfence.Config{
    Capacity:     20,      // Max burst size
    RefillPerSec: 10,      // Tokens added per second
    KeyFunc: func(r *http.Request) string {
        // Custom key extraction (default: IP address)
        if key := r.Header.Get("X-API-Key"); key != "" {
            return key
        }
        return r.RemoteAddr
    },
}
```

## Architecture

```
signalfence/
├── core/           # Token bucket algorithm
├── middleware/     # HTTP middleware
├── store/          # In-memory storage with sync.Map
└── examples/       # Demo server
```

## Future Enhancements

- [ ] Redis backend for distributed systems
- [ ] Sliding window algorithm option
- [ ] Per-route policies
- [ ] Metrics endpoint
- [ ] Prometheus integration

## License

MIT
