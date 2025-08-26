package middleware

import (
	"net/http"
	"time"

	"github.com/rajasatyajit/SupplyChain/internal/auth"
	"github.com/rajasatyajit/SupplyChain/internal/ratelimit"
)

var subscriptionActiveCheck func(ctx interface{}, accountID string) bool

// SetSubscriptionChecker injects a function to check if an account has an active/trialing subscription
func SetSubscriptionChecker(f func(ctx interface{}, accountID string) bool) {
	subscriptionActiveCheck = f
}

// RedisRateQuotaEnforcer uses a Redis-backed manager; if nil, it no-ops and calls next
func RedisRateQuotaEnforcer(m *ratelimit.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if m == nil {
				next.ServeHTTP(w, r)
				return
			}
			p := auth.GetPrincipal(r.Context())
			if p == nil {
				next.ServeHTTP(w, r)
				return
			}
			rpm, monthly := m.PlanLimits(p.PlanCode)
			now := time.Now().UTC()
			method, path := r.Method, r.URL.Path

			// Rate check
			allowed, reset, err := m.CheckRate(r.Context(), p.APIKeyID, method, path, rpm)
			if err == nil && !allowed {
				w.Header().Set("Retry-After", itoaNoAlloc(reset))
				write429(w)
				return
			}

			// Quota pre-check if overage disabled
			if !p.OverageEnabled {
				q, _ := m.GetQuota(r.Context(), p.APIKeyID, now)
				if q >= monthly {
					w.Header().Set("Retry-After", secondsUntil(time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)))
					write429(w)
					return
				}
			}

			// Trial cap: enforce only when there is no active/trialing subscription
			active := false
			if subscriptionActiveCheck != nil {
				active = subscriptionActiveCheck(r.Context(), p.AccountID)
			}
			if !active {
				trialUsed, _ := m.GetTrialUsage(r.Context(), p.AccountID)
				if trialUsed >= 10 {
					w.Header().Set("Retry-After", "3600")
					write429(w)
					return
				}
			}

			// Proceed
			next.ServeHTTP(w, r)

			// Post-increment usage
			_ = m.IncQuota(r.Context(), p.APIKeyID, method, path, now)
			_ = m.IncTrialUsage(r.Context(), p.AccountID)

			// Set headers snapshot
			q, _ := m.GetQuota(r.Context(), p.APIKeyID, now)
			w.Header().Set("X-RateLimit-Limit", itoaNoAlloc(rpm))
			// We can’t reliably get remaining in Redis without another call; set to 0 when exceeded else unknown
			w.Header().Set("X-RateLimit-Remaining", "-")
			w.Header().Set("X-RateLimit-Reset", itoaNoAlloc(reset))

			remaining := monthly - q
			if remaining < 0 {
				remaining = 0
			}
			w.Header().Set("X-Quota-Limit", itoaNoAlloc(monthly))
			w.Header().Set("X-Quota-Remaining", itoaNoAlloc(remaining))
			w.Header().Set("X-Quota-Reset", secondsUntil(time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)))
		})
	}
}
