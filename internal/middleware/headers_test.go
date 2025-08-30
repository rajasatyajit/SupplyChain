package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rajasatyajit/SupplyChain/internal/auth"
)

func TestHeadersPresentAfterRequest(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { time.Sleep(10 * time.Millisecond); w.WriteHeader(200) })
	mw := RateQuotaEnforcer()(h)

	req := httptest.NewRequest("GET", "/v1/alerts", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{AccountID: "acc", APIKeyID: "key", PlanCode: "lite", ClientType: "agent"}))
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("X-RateLimit-Limit") == "" {
		t.Fatalf("missing X-RateLimit-Limit")
	}
	if rec.Header().Get("X-RateLimit-Remaining") == "" {
		t.Fatalf("missing X-RateLimit-Remaining")
	}
	if rec.Header().Get("X-Quota-Limit") == "" {
		t.Fatalf("missing X-Quota-Limit")
	}
	if rec.Header().Get("X-Quota-Remaining") == "" {
		t.Fatalf("missing X-Quota-Remaining")
	}
}
