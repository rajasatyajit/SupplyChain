package api

import (
	"net/http/httptest"
	"testing"
	"time"

	chi "github.com/go-chi/chi/v5"
	"github.com/rajasatyajit/SupplyChain/internal/auth"
	"github.com/rajasatyajit/SupplyChain/internal/store"
)

func TestUsageEndpointsNoRedis(t *testing.T) {
	// Build handler with in-memory store
	st := store.NewInMemoryStore()
	h := NewHandler(st, nil, "", "dev", time.Now().Format(time.RFC3339), "git")
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	// Inject principal
	req := httptest.NewRequest("GET", "/v1/usage", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{AccountID: "acc", APIKeyID: "key", PlanCode: "lite", ClientType: "human"}))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("/v1/usage status %d", rec.Code)
	}
	// ensure JSON includes keys
	if rec.Body.Len() == 0 {
		t.Fatalf("empty body")
	}

	req2 := httptest.NewRequest("GET", "/v1/usage/timeseries?bucket=day", nil)
	req2 = req2.WithContext(auth.WithPrincipal(req2.Context(), &auth.Principal{AccountID: "acc", APIKeyID: "key", PlanCode: "lite", ClientType: "human"}))
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("/v1/usage/timeseries status %d", rec2.Code)
	}
}
