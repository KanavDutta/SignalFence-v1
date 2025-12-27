# SignalFence 🚦

A standalone rate limiting microservice built with Go. Protect your APIs from abuse, spam, and overload with a simple HTTP call.

## What is SignalFence?

SignalFence is a dedicated rate limiting service that sits in front of your APIs. Your application makes a quick HTTP request to SignalFence asking: "Should I allow this request?" SignalFence responds instantly with allow/deny.

**Architecture:**
```
Client → Your API → SignalFence (check) → Allow/Deny → Process or Block
```

**Benefits:**
- ✅ Centralized rate limiting across all your services
- ✅ Language-agnostic (works with any backend)
- ✅ Scales horizontally with Redis
- ✅ Sub-millisecond response times
- ✅ No code changes to your existing APIs

**Protects against:**
- Spam and bots
- Brute-force attacks
- DDoS attempts
- Accidental overload
- Unfair resource usage

## How It Works (Token Bucket Algorithm)

Each client has a "bucket" of tokens:
1. Bucket starts full (e.g., 20 tokens)
2. Each request costs 1 token
3. Tokens refill gradually over time (e.g., 1 token/sec)
4. If bucket is empty → block and tell them when to retry

This is industry-standard, practical, and efficient.

## Quick Start

### Using Docker (Recommended)

```bash
# Start SignalFence + Redis
docker-compose up -d

# SignalFence is now running on http://localhost:8080
```

### Manual Installation

```bash
# Clone the repo
git clone https://github.com/yourusername/signalfence
cd signalfence

# Install dependencies
go mod tidy

# Run with in-memory storage
go run cmd/server/main.go

# Or with Redis
REDIS_ADDR=localhost:6379 go run cmd/server/main.go
```

## API Usage

### Endpoints

#### Check Rate Limit

**Endpoint:** `POST /check`

**Request:**
```json
{
  "client_id": "user-123"
}
```

**Response (Allowed):**
```json
{
  "allowed": true,
  "remaining": 95.0,
  "limit": 100.0,
  "reset_at": 1703620800
}
```

**Response (Blocked):**
```json
{
  "allowed": false,
  "remaining": 0,
  "limit": 100.0,
  "retry_after_ms": 1200,
  "reset_at": 1703620800
}
```

#### Metrics

**Endpoint:** `GET /metrics`

**Response:**
```json
{
  "total_requests": 1523,
  "allowed_requests": 1401,
  "blocked_requests": 122,
  "unique_clients": 45,
  "uptime_seconds": 3600,
  "top_clients": [
    {
      "ClientID": "user-123",
      "TotalRequests": 234,
      "AllowedRequests": 210,
      "BlockedRequests": 24,
      "LastRequestAt": "2024-12-26T10:30:45Z"
    }
  ]
}
```

#### Dashboard

**Endpoint:** `GET /dashboard`

Beautiful real-time dashboard showing:
- Total/allowed/blocked requests
- Unique clients count
- Top 10 most active clients
- Block rates per client
- Auto-refreshes every 2 seconds

### Example: Protecting Your API

**Python:**
```python
import requests

def is_allowed(user_id):
    resp = requests.post('http://localhost:8080/check', 
                         json={'client_id': user_id})
    data = resp.json()
    return data['allowed']

# In your API handler
if not is_allowed(user_id):
    return {"error": "rate_limited"}, 429
```

**Node.js:**
```javascript
async function checkRateLimit(userId) {
  const resp = await fetch('http://localhost:8080/check', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({client_id: userId})
  });
  const data = await resp.json();
  return data.allowed;
}
```

**Go:**
```go
func checkRateLimit(clientID string) (bool, error) {
    body := map[string]string{"client_id": clientID}
    resp, err := http.Post("http://localhost:8080/check", 
                          "application/json", toJSON(body))
    // ... parse response
}
```

## Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run the demo client
go run examples/client-demo/main.go
```

## Advanced Usage

### Custom Rate Limits Per Client

```json
{
  "client_id": "premium-user",
  "capacity": 1000,
  "refill_per_sec": 100
}
```

Different clients can have different limits (e.g., free vs premium users).

## Configuration

### Environment Variables

```bash
PORT=8080                    # Server port (default: 8080)
REDIS_ADDR=localhost:6379    # Redis address (optional, uses in-memory if not set)
REDIS_PASSWORD=secret        # Redis password (optional)
```

### Default Rate Limit Policy

Edit `cmd/server/main.go`:
```go
defaultPolicy := core.Config{
    Capacity:     100,  // 100 requests burst
    RefillPerSec: 10,   // 10 requests per second sustained
}
```

### Deployment Options

**Single Server (In-Memory):**
- Fast, simple
- Good for development or single-instance deployments
- Limits reset on restart

**Multi-Server (Redis):**
- Distributed rate limiting
- Limits shared across all instances
- Production-ready
- Survives restarts

## Architecture

```
signalfence/
├── cmd/server/         # Main service entry point
├── api/                # HTTP handlers
├── core/               # Token bucket algorithm
├── store/              # Storage backends (memory, Redis)
├── examples/           # Client demos
├── Dockerfile          # Container image
└── docker-compose.yml  # Full stack deployment
```

**How It Works:**
1. Client makes request to your API
2. Your API calls SignalFence: `POST /check` with `client_id`
3. SignalFence checks token bucket for that client
4. Returns `allowed: true/false` in <1ms
5. Your API proceeds or returns 429

## Features

- ✅ Standalone microservice (language-agnostic)
- ✅ Token bucket algorithm
- ✅ In-memory + Redis storage
- ✅ Custom policies per client
- ✅ Real-time metrics & dashboard
- ✅ Docker + docker-compose ready
- ✅ Comprehensive tests
- ✅ Sub-millisecond latency

## Monitoring & Observability

SignalFence includes built-in metrics and a real-time dashboard:

**Dashboard:** Open `http://localhost:8080/dashboard` in your browser
- Real-time statistics
- Top clients by request volume
- Block rates and patterns
- Auto-refreshes every 2 seconds

**Metrics API:** `GET /metrics` returns JSON for integration with monitoring tools

## Production Checklist

- [ ] Deploy with Redis for multi-instance setups
- [ ] Set up monitoring (health endpoint at `/health`)
- [ ] Configure appropriate default limits
- [ ] Add authentication for the `/check` endpoint
- [ ] Export metrics to Prometheus/Grafana
- [ ] Use HTTPS in production

## Future Enhancements

- [ ] Admin API (reset limits, update policies)
- [ ] Prometheus metrics export
- [ ] Sliding window algorithm option
- [ ] Weighted costs (some requests cost more tokens)
- [ ] Grafana dashboard templates
- [ ] Alert webhooks (notify when clients are blocked)

## License

MIT
