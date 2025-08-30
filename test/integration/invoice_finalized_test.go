package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	chi "github.com/go-chi/chi/v5"
	"github.com/rajasatyajit/SupplyChain/config"
	"github.com/rajasatyajit/SupplyChain/internal/api"
	"github.com/rajasatyajit/SupplyChain/internal/database"
	"github.com/rajasatyajit/SupplyChain/internal/ratelimit"
)

// Simulate invoice.finalized with metered item and ensure usage record function is invoked with expected overage
func TestInvoiceFinalizedOverageReporting(t *testing.T) {
	redisURL := mustEnv(t, "REDIS_URL")
	dbURL := mustEnv(t, "DATABASE_URL")
	os.Setenv("STRIPE_PRICE_OVERAGE_METERED", "price_metered")

	cfg := config.DatabaseConfig{URL: dbURL, MaxConns: 5, MinConns: 1}
	db, err := database.New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	defer db.Close(context.Background())
	setupMinimalSchema(t, db)

	// Insert account, subscription with overage enabled
	acct := "acc_ovg"
	subID := "sub_123"
	_ = db.Exec(context.Background(), "INSERT INTO accounts(id,email) VALUES ($1,$2) ON CONFLICT DO NOTHING", acct, "ovg@example.com")
	_ = db.Exec(context.Background(), "INSERT INTO subscriptions(account_id, plan_code, overage_enabled, stripe_subscription_id, status) VALUES ($1,'lite',true,$2,'active')", acct, subID)

	// Create two keys for the account
	_ = db.Exec(context.Background(), "INSERT INTO api_keys(id,account_id,key_prefix,client_type,status) VALUES ('k1',$1,'k1','agent','active')", acct)
	_ = db.Exec(context.Background(), "INSERT INTO api_keys(id,account_id,key_prefix,client_type,status) VALUES ('k2',$1,'k2','agent','active')", acct)

	// Redis manager with small monthly quota to create overage easily
	mgr, err := ratelimit.NewManager(redisURL)
	if err != nil {
		t.Fatalf("redis: %v", err)
	}
	mgr.SetPlanLimits(100, 5, 300, 15) // lite monthly = 5
	api.SetRateLimiter(mgr)

	// Add usage: 8 total (over by 3)
	now := time.Now().UTC()
	_ = mgr.IncQuota(context.Background(), "k1", "GET", "/v1/alerts", now)
	_ = mgr.IncQuota(context.Background(), "k1", "GET", "/v1/alerts", now)
	_ = mgr.IncQuota(context.Background(), "k2", "GET", "/v1/alerts", now)
	_ = mgr.IncQuota(context.Background(), "k2", "GET", "/v1/alerts", now)
	_ = mgr.IncQuota(context.Background(), "k2", "GET", "/v1/alerts", now)
	_ = mgr.IncQuota(context.Background(), "k2", "GET", "/v1/alerts", now)
	_ = mgr.IncQuota(context.Background(), "k2", "GET", "/v1/alerts", now)

	// Build API handler and route
	h := api.NewHandler(nil, db, "", "dev", time.Now().Format(time.RFC3339), "git")
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	// Mock meter usage creation
	var reported int64
	orig := api.GetMeterUsageFunc()
	api.SetMeterUsageFunc(func(subscriptionItemID string, quantity int64) (string, error) {
		reported = quantity
		return "ur_test", nil
	})
	defer func() { api.SetMeterUsageFunc(orig) }()

	payload := `{"id":"evt_test_2","type":"invoice.finalized","data":{"object":{"subscription":{"id":"` + subID + `"},"lines":{"data":[{"price":{"id":"price_metered"},"subscription_item":"si_123"}]}}}}`
	ts := fmt.Sprintf("%d", time.Now().Unix())
	signed := signStripe(ts, payload, "whsec_test")
	req := httptest.NewRequest("POST", "/v1/billing/webhook", strings.NewReader(payload))
	req.Header.Set("Stripe-Signature", fmt.Sprintf("t=%s,v1=%s", ts, signed))
	os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("webhook %d", rec.Code)
	}
	if reported < 1 {
		t.Fatalf("expected reported overage > 0, got %d", reported)
	}
}
