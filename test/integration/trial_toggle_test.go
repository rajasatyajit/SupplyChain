package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	chi "github.com/go-chi/chi/v5"
	"github.com/rajasatyajit/SupplyChain/config"
	"github.com/rajasatyajit/SupplyChain/internal/auth"
	"github.com/rajasatyajit/SupplyChain/internal/database"
	middlewares "github.com/rajasatyajit/SupplyChain/internal/middleware"
)

// Ensure that when subscription is trialing/active, the trial cap does not block after 10 requests
func TestTrialCapDisabledWhenSubscriptionActive(t *testing.T) {
	dbURL := mustEnv(t, "DATABASE_URL")
	cfg := config.DatabaseConfig{URL: dbURL, MaxConns: 5, MinConns: 1}
	db, err := database.New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	defer db.Close(context.Background())
	setupMinimalSchema(t, db)

	// Insert subscription trialing
	accountID := "acc_trial"
	_ = db.Exec(context.Background(), "INSERT INTO subscriptions(account_id, plan_code, status) VALUES ($1,'lite','trialing')", accountID)

	// Build a tiny router with only middleware and dummy endpoint
	r := chi.NewRouter()
	// Inject subscription checker against this DB
	middlewares.SetSubscriptionChecker(func(ctx interface{}, acct string) bool {
		row := db.QueryRow(context.Background(), "SELECT 1 FROM subscriptions WHERE account_id=$1 AND status IN ('active','trialing')", acct)
		var one int
		if s, ok := row.(interface{ Scan(dest ...any) error }); ok {
			return s.Scan(&one) == nil
		}
		return false
	})
	r.Use(middlewares.RateQuotaEnforcer()) // in-memory enforcer is fine for this test
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	r.Method("GET", "/v1/alerts", h)

	p := &auth.Principal{AccountID: accountID, APIKeyID: "key", PlanCode: "lite", ClientType: "agent", OverageEnabled: false}
	var last int
	for i := 0; i < 12; i++ {
		req := httptest.NewRequest("GET", "/v1/alerts", nil)
		req = req.WithContext(auth.WithPrincipal(req.Context(), p))
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		last = rec.Code
		if last != 200 {
			t.Fatalf("unexpected status at req %d: %d", i+1, last)
		}
	}
}

// removed testCtx helper; use context.Background() in tests
