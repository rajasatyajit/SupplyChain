package api

import "github.com/rajasatyajit/SupplyChain/internal/ratelimit"

// SetRateLimiter allows main to inject the limiter for usage endpoints
func SetRateLimiter(m *ratelimit.Manager) { setRateLimiter(m) }
