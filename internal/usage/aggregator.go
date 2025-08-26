package usage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rajasatyajit/SupplyChain/internal/auth"
	"github.com/rajasatyajit/SupplyChain/internal/database"
	"github.com/rajasatyajit/SupplyChain/internal/logger"
	"github.com/rajasatyajit/SupplyChain/internal/ratelimit"
)

// StartAggregator periodically flushes Redis usage into Postgres usage_aggregates
func StartAggregator(ctx context.Context, db *database.DB, rl *ratelimit.Manager) {
	if db == nil || !db.IsConfigured() || rl == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				FlushOnce(ctx, db, rl)
			}
		}
	}()
}

// FlushOnce exposes a single aggregation cycle for tests and ops
func FlushOnce(ctx context.Context, db *database.DB, rl *ratelimit.Manager) {
	repo := auth.NewRepository(db)
	pairs, err := repo.ListAllActiveAPIKeys(ctx)
	if err != nil {
		logger.Error("usage flush: list keys failed", "error", err)
		return
	}
	now := time.Now().UTC()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)
	for _, p := range pairs {
		total, err := rl.GetQuota(ctx, p.KeyID, now)
		if err != nil {
			continue
		}
		ep, _ := rl.ListEndpointUsage(ctx, p.KeyID, now)
		b, _ := json.Marshal(ep)
		_ = db.Exec(ctx, `
			INSERT INTO usage_aggregates(account_id, api_key_id, period_start, period_end, total_requests, per_endpoint)
			VALUES ($1,$2,$3,$4,$5,$6)
			ON CONFLICT(account_id, api_key_id, period_start, period_end)
			DO UPDATE SET total_requests=EXCLUDED.total_requests, per_endpoint=EXCLUDED.per_endpoint
		`, p.AccountID, p.KeyID, periodStart, periodEnd, total, string(b))
	}
}
