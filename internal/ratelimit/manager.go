package ratelimit

import (
	"context"
	"fmt"
	"strings"
	"time"

	redis "github.com/redis/go-redis/v9"
)

// Manager provides Redis-backed rate limiting and quota accounting
type Manager struct {
	redis *redis.Client
	// monthly limits per plan (temporary; load from DB later)
	liteRPM          int
	liteMonthlyQuota int
	proRPM           int
	proMonthlyQuota  int
}

// SetPlanLimits allows tests to override plan limits
func (m *Manager) SetPlanLimits(liteRPM, liteMonthly, proRPM, proMonthly int) {
	m.liteRPM = liteRPM
	m.liteMonthlyQuota = liteMonthly
	m.proRPM = proRPM
	m.proMonthlyQuota = proMonthly
}

func NewManager(redisURL string) (*Manager, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	client := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &Manager{redis: client, liteRPM: 20, liteMonthlyQuota: 450000, proRPM: 60, proMonthlyQuota: 1350000}, nil
}

func (m *Manager) Close() error { return m.redis.Close() }

// PlanLimits returns rpm and monthly quota for given plan code
func (m *Manager) PlanLimits(planCode string) (rpm int, monthly int) {
	if strings.ToLower(planCode) == "pro" {
		return m.proRPM, m.proMonthlyQuota
	}
	return m.liteRPM, m.liteMonthlyQuota
}

// Keys helpers
func monthKey(t time.Time) string { return t.Format("200601") }

// CheckRate returns allowed=false if rate bucket exhausted; it also returns reset seconds
func (m *Manager) CheckRate(ctx context.Context, apiKeyID, method, path string, rpm int) (allowed bool, resetSec int, err error) {
	now := time.Now().UTC()
	window := now.Unix() / 60 // minute window
	rk := fmt.Sprintf("rl:%s:%s:%s:%d", apiKeyID, method, path, window)
	// Use INCR and set TTL if first time
	pipe := m.redis.TxPipeline()
	incr := pipe.Incr(ctx, rk)
	pipe.Expire(ctx, rk, time.Minute)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return false, 0, err
	}
	count := int(incr.Val())
	if count > rpm {
		// compute seconds until window end
		secPassed := int(now.Unix() % 60)
		return false, 60 - secPassed, nil
	}
	return true, 0, nil
}

// GetQuota returns current total for this period
func (m *Manager) GetQuota(ctx context.Context, apiKeyID string, now time.Time) (int, error) {
	qk := fmt.Sprintf("quota:%s:%s:total", apiKeyID, monthKey(now))
	val, err := m.redis.Get(ctx, qk).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

// IncQuota increments totals after successful request
func (m *Manager) IncQuota(ctx context.Context, apiKeyID, method, path string, now time.Time) error {
	totalKey := fmt.Sprintf("quota:%s:%s:total", apiKeyID, monthKey(now))
	epKey := fmt.Sprintf("quota:%s:%s:ep:%s:%s", apiKeyID, monthKey(now), method, path)
	exp := time.Duration(time.Until(time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)))
	pipe := m.redis.TxPipeline()
	pipe.Incr(ctx, totalKey)
	pipe.Expire(ctx, totalKey, exp)
	pipe.Incr(ctx, epKey)
	pipe.Expire(ctx, epKey, exp)
	_, err := pipe.Exec(ctx)
	return err
}

// GetEndpointQuota returns per-endpoint usage for this period
func (m *Manager) GetEndpointQuota(ctx context.Context, apiKeyID, method, path string, now time.Time) (int, error) {
	epKey := fmt.Sprintf("quota:%s:%s:ep:%s:%s", apiKeyID, monthKey(now), method, path)
	val, err := m.redis.Get(ctx, epKey).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

// Trial usage
func (m *Manager) GetTrialUsage(ctx context.Context, accountID string) (int, error) {
	k := fmt.Sprintf("trial:%s:used", accountID)
	val, err := m.redis.Get(ctx, k).Int()
	if err == redis.Nil { return 0, nil }
	return val, err
}
func (m *Manager) IncTrialUsage(ctx context.Context, accountID string) error {
	k := fmt.Sprintf("trial:%s:used", accountID)
return m.redis.Incr(ctx, k).Err()
}

// ListEndpointUsage scans Redis for endpoint counters for a key in current month
func (m *Manager) ListEndpointUsage(ctx context.Context, apiKeyID string, now time.Time) (map[string]int, error) {
	res := make(map[string]int)
	pattern := fmt.Sprintf("quota:%s:%s:ep:*", apiKeyID, monthKey(now))
	var cursor uint64
	for {
		keys, cur, err := m.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil { return nil, err }
		cursor = cur
		for _, k := range keys {
			v, err := m.redis.Get(ctx, k).Int()
			if err == nil {
				// k format: quota:<key>:<month>:ep:<METHOD>:<PATH>
				parts := strings.Split(k, ":ep:")
				if len(parts) == 2 {
					res[parts[1]] = v
				}
			}
		}
		if cursor == 0 { break }
	}
	return res, nil
}

// SumQuotas returns the sum of totals for a set of api key IDs for current month
func (m *Manager) SumQuotas(ctx context.Context, apiKeyIDs []string, now time.Time) (int, error) {
	total := 0
	for _, id := range apiKeyIDs {
		q, err := m.GetQuota(ctx, id, now)
		if err != nil { return 0, err }
		total += q
	}
	return total, nil
}
