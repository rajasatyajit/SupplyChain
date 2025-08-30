package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	chi "github.com/go-chi/chi/v5"
	"github.com/rajasatyajit/SupplyChain/internal/store"
)

// invalid signature should return 400
func TestStripeWebhookInvalidSignature(t *testing.T) {
	st := store.NewInMemoryStore()
	h := NewHandler(st, nil, "", "dev", time.Now().Format(time.RFC3339), "git")
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	req := httptest.NewRequest("POST", "/v1/billing/webhook", strings.NewReader("{}"))
	req.Header.Set("Stripe-Signature", "invalid")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
