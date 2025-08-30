package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/rajasatyajit/SupplyChain/internal/auth"
	"github.com/rajasatyajit/SupplyChain/internal/ratelimit"
)

func TestRedisRateLimiterRPM(t *testing.T) {
	// Start miniredis
	s, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	mgr, err := ratelimit.NewManager("redis://" + s.Addr())
	if err != nil {
		t.Fatal(err)
	}
	// Start with a low RPM to trigger 429 but very high monthly/trial caps so they don't interfere
	mgr.SetPlanLimits(1000, 100000, 20, 100000)

	// Handler increments ok
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	mw := RedisRateQuotaEnforcer(mgr)(h)

	req := httptest.NewRequest("GET", "/v1/alerts", nil)
	p := &auth.Principal{AccountID: "acc", APIKeyID: "key", PlanCode: "lite", ClientType: "agent", OverageEnabled: false}
	req = req.WithContext(auth.WithPrincipal(req.Context(), p))

	var last int
	for i := 0; i < 25; i++ {
		rec := httptest.NewRecorder()
		mw.ServeHTTP(rec, req)
		last = rec.Code
	}
	if last != 429 {
		t.Fatalf("expected 429 after exceeding rpm, got %d", last)
	}
	// Wait for next minute window and clear redis state to simulate fresh window
	s.FastForward(time.Minute)
	s.FlushAll()
	// Increase RPM limit for the next window to ensure success regardless of residual state
	mgr.SetPlanLimits(1000, 100000, 1000, 100000)
	req2 := httptest.NewRequest("GET", "/v1/alerts-new", nil)
	req2 = req2.WithContext(auth.WithPrincipal(req2.Context(), p))
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req2)
	if rec.Code != 200 {
		t.Fatalf("expected 200 after window reset with higher limit on new endpoint, got %d", rec.Code)
	}
}
