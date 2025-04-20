# ratelimiter

A high-performance, fully testable, token-bucket-based rate limiter for Go. Designed to be lightweight, flexible/ this library supports burstable rate limiting, precise token accounting, and deterministic testing using a pluggable clock.
---

## 🚀 

- ✅ **Custom Clock Interface** – Enables deterministic time manipulation for testing and simulations.
- 🎯 **Token Accounting Precision** – Tracks available tokens with float accuracy for smoother and more granular control.
- 🧪 **Unit Test Coverage** – Every code path is covered with tests, thanks to a fake clock and clean design.
- 🪄 **Reservation & Cancellation APIs** – Make future reservations for tokens and optionally cancel them with rollback.
- 🧵 **Thread-Safe by Design** – Uses mutex-based synchronization for safe concurrent usage.
- 📦 **Minimal & Self-contained** – Zero third-party dependencies.

---

## 📦 APIs

| API                             | Description                                                                  |
|---------------------------------|------------------------------------------------------------------------------|
| `New(rate Rate, burst int, clk Clock)` | Initialize a new rate limiter with custom rate and clock       |
| `Allow()`                       | Returns `true` if one token can be consumed immediately                      |
| `AllowN(n int)`                 | Returns `true` if `n` tokens can be consumed immediately                     |
| `Reserve()`                     | Reserve one token for future use (non-blocking)                              |
| `ReserveN(n int)`              | Reserve `n` tokens with a delay and cancel capability                        |
| `Wait(n int)`                  | Block until `n` tokens are available or return an error                      |
| `SetRate(rate Rate)`           | Dynamically update token generation rate                                     |
| `SetBurst(burst int)`          | Dynamically update burst capacity                                            |
| `Rate()`                        | Returns the current rate of token generation                                 |
| `Burst()`                       | Returns the current burst size                                               |
---

---

## ⏱️ Token Bucket Algorithm

This limiter stores tokens in a bucket up to a `burst` size. Tokens are added at a fixed rate over time, and requests consume tokens.

---