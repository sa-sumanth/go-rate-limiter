# go-rate-limiter

A production-inspired rate limiter built in Go using Gin, Redis, and Docker Compose.
Implements five rate limiting algorithms with Redis-backed atomic operations via Lua scripts.

## Stack
- **Go + Gin** — HTTP server and middleware chain
- **Redis** — shared backing store for all algorithms
- **Docker Compose** — runs the app and Redis together
- **pprof** — profiling endpoints for performance analysis

## Running

    docker compose up --build

App runs on `http://localhost:8080`

## Algorithms
- Token Bucket
- Leaky Bucket
- Fixed Window
- Sliding Window Log
- Sliding Window Counter

## Endpoints

| Endpoint | Description |
|---|---|
| `GET /health` | Health check |
| `GET /metrics` | Metrics stub (Prometheus coming soon) |
| `GET /token_bucket` | Token bucket rate limited |
| `GET /leaky_bucket` | Leaky bucket rate limited |
| `GET /fixed_window` | Fixed window rate limited |
| `GET /sliding_window_log` | Sliding window log rate limited |
| `GET /sliding_window_counter` | Sliding window counter rate limited |
| `GET /debug/pprof/*` | pprof profiling |

## Why Redis + Lua Scripts
All state lives in Redis so the limiter works correctly across multiple app instances.
Each algorithm runs as an atomic Lua script — eliminating race conditions without distributed locks.
