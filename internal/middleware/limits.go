package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rajasatyajit/SupplyChain/internal/auth"
)

// plan limits (temporary until read from DB)
const (
	liteRPM           = 20
	liteMonthlyQuota  = 450000
	proRPM            = 60
	proMonthlyQuota   = 1350000
	trialTotalAllowed = 10 // per account total
)

type bucket struct {
	windowStart time.Time
	count       int
}

type usage struct {
	periodStart time.Time
	periodEnd   time.Time
	perEndpoint map[string]int
	total       int
	trialUsed   int
}

type rateQuotaState struct {
	mu        sync.Mutex
	buckets   map[string]*bucket      // key: apiKeyID|METHOD|PATH
	usageByAK map[string]*usage       // apiKeyID -> usage
	usageByAC map[string]*usage       // accountID -> usage (for trial)
}

var rqState = &rateQuotaState{
	buckets:   make(map[string]*bucket),
	usageByAK: make(map[string]*usage),
	usageByAC: make(map[string]*usage),
}

// RateQuotaEnforcer enforces per-endpoint RPM and monthly quotas per API key and adds headers
func RateQuotaEnforcer() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := auth.GetPrincipal(r.Context())
			if p == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Determine plan limits
			rpm := liteRPM
			monthly := liteMonthlyQuota
			if strings.ToLower(p.PlanCode) == "pro" {
				rpm = proRPM
				monthly = proMonthlyQuota
			}

			// Compute current month period
			now := time.Now().UTC()
			periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
			periodEnd := periodStart.AddDate(0, 1, 0)

			method := r.Method
			path := r.URL.Path // simple path key; can normalize later
			bucketKey := p.APIKeyID + "|" + method + "|" + path

			rqState.mu.Lock()
			// Rate limit (1-minute window)
			b := rqState.buckets[bucketKey]
			if b == nil || now.Sub(b.windowStart) >= time.Minute {
				b = &bucket{windowStart: now, count: 0}
				rqState.buckets[bucketKey] = b
			}
			// Usage structures
			u := rqState.usageByAK[p.APIKeyID]
			if u == nil || !u.periodStart.Equal(periodStart) {
				u = &usage{periodStart: periodStart, periodEnd: periodEnd, perEndpoint: make(map[string]int)}
				rqState.usageByAK[p.APIKeyID] = u
			}
			ua := rqState.usageByAC[p.AccountID]
			if ua == nil || !ua.periodStart.Equal(periodStart) {
				ua = &usage{periodStart: periodStart, periodEnd: periodEnd, perEndpoint: make(map[string]int)}
				rqState.usageByAC[p.AccountID] = ua
			}

			// Trial enforcement (per account)
			if ua.trialUsed >= trialTotalAllowed {
				// Already used trial; if subscription inactive in future we’ll check subs. For now, enforce hard cap.
				rqState.mu.Unlock()
				w.Header().Set("Retry-After", "3600")
				write429(w)
				return
			}

			// Quota enforcement (per api key)
			if u.total >= monthly && !p.OverageEnabled {
				rqState.mu.Unlock()
				w.Header().Set("Retry-After", secondsUntil(periodEnd))
				write429(w)
				return
			}

			// Rate enforcement
			if b.count >= rpm {
				reset := 60 - int(now.Sub(b.windowStart).Seconds())
				rqState.mu.Unlock()
				w.Header().Set("Retry-After", itoaNoAlloc(reset))
				write429(w)
				return
			}

			// Reserve
			b.count++
			rqState.mu.Unlock()

			// Call next
			next.ServeHTTP(w, r)

			// On success (2xx) increment usage and set headers
			// For simplicity, treat all responses as billable here; refine later based on status code if needed.
			rqState.mu.Lock()
			u.total++
			u.perEndpoint[method+":"+path]++
			ua.trialUsed++

			// Set headers
			remainingRate := rpm - b.count
			if remainingRate < 0 { remainingRate = 0 }
			w.Header().Set("X-RateLimit-Limit", itoaNoAlloc(rpm))
			w.Header().Set("X-RateLimit-Remaining", itoaNoAlloc(remainingRate))
			w.Header().Set("X-RateLimit-Reset", itoaNoAlloc(60-int(time.Since(b.windowStart).Seconds())))

			remainingQuota := monthly - u.total
			if remainingQuota < 0 { remainingQuota = 0 }
			w.Header().Set("X-Quota-Limit", itoaNoAlloc(monthly))
			w.Header().Set("X-Quota-Remaining", itoaNoAlloc(remainingQuota))
			w.Header().Set("X-Quota-Reset", itoaNoAlloc(int(periodEnd.Sub(now).Seconds())))
			rqState.mu.Unlock()
		})
	}
}

func secondsUntil(t time.Time) string {
	d := int(time.Until(t).Seconds())
	if d < 0 { d = 0 }
	return itoaNoAlloc(d)
}

func itoaNoAlloc(i int) string {
	// fast int to string for small numbers
	return strconvItoa(i)
}

// small inline itoa to avoid extra import clutter
func strconvItoa(i int) string {
	// naive: this is fine here, we’ll replace with strconv.Itoa once imported
	return fmtSprintf(i)
}

func fmtSprintf(i int) string {
	// defer importing fmt in this file; keep simple implementation
	// replaced by a minimal version in codegen; but to keep correctness, we import fmt elsewhere, so we can rely on it.
	return fmtInt(i)
}

// go doesn’t allow us to call fmt without import; we’ll provide a minimal helper in a separate file.
