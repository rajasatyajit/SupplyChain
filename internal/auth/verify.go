package auth

import (
	"context"
	"net/http"

	"github.com/rajasatyajit/SupplyChain/internal/database"
)

// VerifyAPIKey is a placeholder that accepts any non-empty key.
// It returns a synthetic principal based on headers for progressive rollout.
// TODO: Replace with real DB/Redis lookup and validation logic.
func VerifyAPIKey(r *http.Request, key string, clientType string) (*Principal, error) {
	if key == "" {
		return nil, ErrUnauthorized
	}
	if clientType != "agent" && clientType != "human" && clientType != "" {
		return nil, ErrUnauthorized
	}
	// Attempt DB-backed lookup if available
	ctx := r.Context()
	if dbCtx := ctx.Value(dbKey{}); dbCtx != nil {
		if db, ok := dbCtx.(*database.DB); ok && db != nil && db.IsConfigured() {
			repo := NewRepository(db)
			rec, err := repo.LookupAPIKey(ctx, key)
			if err == nil && rec != nil {
				return &Principal{
					AccountID:      rec.AccountID,
					APIKeyID:       rec.APIKeyID,
					PlanCode:       rec.PlanCode,
					ClientType:     rec.ClientType,
					OverageEnabled: rec.OverageEnabled,
				}, nil
			}
		}
	}
	// Fallback permissive principal for progressive rollout
	p := &Principal{
		AccountID:      "acc_dev",
		APIKeyID:       "key_dev",
		PlanCode:       "lite",
		ClientType:     clientType,
		OverageEnabled: false,
	}
	return p, nil
}

type dbKey struct{}

// WithDB attaches the database to request context for auth lookups
func WithDB(ctx context.Context, db *database.DB) context.Context { return context.WithValue(ctx, dbKey{}, db) }
