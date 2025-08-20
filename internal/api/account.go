package api

import "sync"

var rlOnce sync.Once
var rlGlobal *ratelimit.Manager

func setRateLimiter(m *ratelimit.Manager) { rlOnce.Do(func(){ rlGlobal = m }) }
func getRateLimiter() *ratelimit.Manager { return rlGlobal }

import (
	"net/http"
	"time"

	"github.com/rajasatyajit/SupplyChain/internal/auth"
	"github.com/rajasatyajit/SupplyChain/internal/ratelimit"
)

// meHandler returns plan and period info for the calling API key
func (h *Handler) meHandler(w http.ResponseWriter, r *http.Request) {
	p := auth.GetPrincipal(r.Context())
	if p == nil {
		h.writeErrorResponse(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	// Placeholder period boundaries (monthly calendar)
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	resp := map[string]any{
		"account_id": p.AccountID,
		"plan":       p.PlanCode,
		"overage_enabled": p.OverageEnabled,
		"period_start":    start,
		"period_end":      end,
	}
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// limitsHandler returns nominal per-endpoint limits (placeholder values)
func (h *Handler) limitsHandler(w http.ResponseWriter, r *http.Request) {
	p := auth.GetPrincipal(r.Context())
	if p == nil {
		h.writeErrorResponse(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	// Placeholder limits; will be loaded from plan entitlements later
	limits := map[string]any{
		"per_minute": map[string]int{
			"/v1/alerts:GET": func() int { if p.PlanCode == "pro" { return 60 } ; return 20 }(),
		},
		"monthly_quota": func() int { if p.PlanCode == "pro" { return 1350000 } ; return 450000 }(),
	}
	h.writeJSONResponse(w, http.StatusOK, limits)
}

// usageHandler returns current usage summary reading from Redis if available
func (h *Handler) usageHandler(w http.ResponseWriter, r *http.Request) {
	p := auth.GetPrincipal(r.Context())
	if p == nil {
		h.writeErrorResponse(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	// If Redis limiter is available, fetch totals and per-endpoint; otherwise return zeros
	var total int
	var perEndpoint map[string]int
	if mgr := getRateLimiter(); mgr != nil {
		q, _ := mgr.GetQuota(r.Context(), p.APIKeyID, start)
		total = q
		perEndpoint, _ = mgr.ListEndpointUsage(r.Context(), p.APIKeyID, start)
	}
	resp := map[string]any{
		"account_id":  p.AccountID,
		"period_start": start,
		"period_end":   start.AddDate(0, 1, 0),
		"total":        total,
		"per_endpoint": perEndpoint,
	}
	h.writeJSONResponse(w, http.StatusOK, resp)
}
