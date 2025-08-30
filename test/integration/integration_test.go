package integration

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/rajasatyajit/SupplyChain/internal/usage"
)

func mustEnv(t *testing.T, k string) string {
	t.Helper()
	v := os.Getenv(k)
	if v == "" {
		t.Skipf("env %s not set; skipping integration", k)
	}
	return v
}

func setupMinimalSchema(t *testing.T, db *database.DB) {
	sqls := []string{
		"CREATE TABLE IF NOT EXISTS accounts (id TEXT PRIMARY KEY, email TEXT);",
		"CREATE TABLE IF NOT EXISTS api_keys (id TEXT, account_id TEXT, key_prefix TEXT, key_hash BYTEA, client_type TEXT, status TEXT);",
		"CREATE TABLE IF NOT EXISTS usage_aggregates (account_id TEXT, api_key_id TEXT, period_start TIMESTAMPTZ, period_end TIMESTAMPTZ, total_requests BIGINT, per_endpoint JSONB, UNIQUE(account_id, api_key_id, period_start, period_end));",
		"CREATE TABLE IF NOT EXISTS webhook_events (id TEXT PRIMARY KEY, type TEXT, payload JSONB, received_at TIMESTAMPTZ DEFAULT now());",
		"CREATE TABLE IF NOT EXISTS processed_events (event_id TEXT PRIMARY KEY, processed_at TIMESTAMPTZ DEFAULT now());",
		"CREATE TABLE IF NOT EXISTS subscriptions (account_id TEXT, plan_code TEXT, overage_enabled BOOLEAN, stripe_customer_id TEXT, stripe_subscription_id TEXT, status TEXT, current_period_start TIMESTAMPTZ, current_period_end TIMESTAMPTZ);",
	}
	ctx := context.Background()
	for _, s := range sqls {
		if err := db.Exec(ctx, s); err != nil {
			t.Fatalf("schema exec: %v", err)
		}
	}
}

func TestAggregatorFlushesToDB(t *testing.T) {
	redisURL := mustEnv(t, "REDIS_URL")
	dbURL := mustEnv(t, "DATABASE_URL")
	cfg := config.DatabaseConfig{URL: dbURL, MaxConns: 5, MinConns: 1}
	db, err := database.New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	defer db.Close(context.Background())
	setupMinimalSchema(t, db)

	mgr, err := ratelimit.NewManager(redisURL)
	if err != nil {
		t.Fatalf("redis: %v", err)
	}

	// Insert account and key
	ctx := context.Background()
	acct := "acc_it"
	key := "key_it"
	_ = db.Exec(ctx, "INSERT INTO accounts(id,email) VALUES ($1,$2) ON CONFLICT DO NOTHING", acct, "it@example.com")
	_ = db.Exec(ctx, "INSERT INTO api_keys(id,account_id,key_prefix,client_type,status) VALUES ($1,$2,$3,'agent','active')", key, acct, key)

	// Increment some usage in Redis
	now := time.Now().UTC()
	_ = mgr.IncQuota(ctx, key, "GET", "/v1/alerts", now)
	_ = mgr.IncQuota(ctx, key, "GET", "/v1/alerts", now)

	// Flush once
	usage.FlushOnce(ctx, db, mgr)

	// Verify row exists
	row := db.QueryRow(ctx, "SELECT total_requests FROM usage_aggregates WHERE account_id=$1 AND api_key_id=$2", acct, key)
	var total int
	if s, ok := row.(interface{ Scan(dest ...any) error }); ok {
		if err := s.Scan(&total); err != nil {
			t.Fatalf("scan: %v", err)
		}
	}
	if total < 2 {
		t.Fatalf("expected at least 2, got %d", total)
	}
}

func TestStripeWebhookProcessedEvent(t *testing.T) {
	dbURL := mustEnv(t, "DATABASE_URL")
	cfg := config.DatabaseConfig{URL: dbURL, MaxConns: 5, MinConns: 1}
	db, err := database.New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	defer db.Close(context.Background())
	setupMinimalSchema(t, db)

	st := api.NewHandler(nil, db, "", "dev", time.Now().Format(time.RFC3339), "git")
	r := chi.NewRouter()
	st.RegisterRoutes(r)

	secret := "whsec_test"
	os.Setenv("STRIPE_WEBHOOK_SECRET", secret)
	payload := `{"id":"evt_test_1","type":"customer.subscription.deleted","data":{"object":{}}}`
	ts := fmt.Sprintf("%d", time.Now().Unix())
	signed := signStripe(ts, payload, secret)
	req := httptest.NewRequest("POST", "/v1/billing/webhook", strings.NewReader(payload))
	req.Header.Set("Stripe-Signature", fmt.Sprintf("t=%s,v1=%s", ts, signed))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("webhook %d", rec.Code)
	}

	// Confirm processed_events has record
	row := db.QueryRow(context.Background(), "SELECT 1 FROM processed_events WHERE event_id=$1", "evt_test_1")
	var one int
	if s, ok := row.(interface{ Scan(dest ...any) error }); ok {
		if err := s.Scan(&one); err != nil {
			t.Fatalf("processed not recorded: %v", err)
		}
	}
}

func signStripe(ts, payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts))
	mac.Write([]byte("."))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
