# SignalFence 🛡️
A lightweight **rate-limiting + abuse-protection** layer for backend APIs. SignalFence enforces fair usage using a **token bucket** algorithm, allowing controlled bursts while protecting services from overload and spam.

---

## Why SignalFence
Most student projects focus on CRUD features. SignalFence focuses on a production-style problem: **protecting APIs**.

SignalFence helps you:
- prevent request flooding (accidental or malicious)
- reduce brute-force attempts on sensitive endpoints
- keep performance stable under burst traffic
- apply consistent rate limits per **API key** or **IP**

---

## How it works (Token Bucket)
Each client (API key/IP) has a “bucket” of tokens.

- The bucket starts full (e.g., `capacity = 20`)
- Each request consumes `1` token
- Tokens refill over time (e.g., `refillPerSec = 1`)
- If the bucket is empty, SignalFence returns **429 Too Many Requests**
  and tells the client when to retry.

This approach supports **burst traffic** while maintaining a stable long-term rate.

---

## Features
### v1 (MVP)
- ✅ Token bucket rate limiter
- ✅ Limits by **API key** (fallback to IP)
- ✅ Express middleware integration
- ✅ Standard rate limit headers:
  - `X-RateLimit-Limit`
  - `X-RateLimit-Remaining`
  - `Retry-After`
- ✅ Unit tests for core scheduling logic (correctness + edge cases)

### Planned / Roadmap
- 🔜 Redis-backed store (persistence + multi-instance support)
- 🔜 Per-route policies (different limits per endpoint)
- 🔜 Metrics dashboard / `/metrics` endpoint
- 🔜 Sliding window limiter option

---

## Tech Stack
- **Node.js + TypeScript**
- **Express** (demo + middleware)
- **Unit tests** (Jest/Vitest)

---

## Project Structure
