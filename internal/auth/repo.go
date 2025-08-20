package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/rajasatyajit/SupplyChain/internal/database"
	"golang.org/x/crypto/bcrypt"
)

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

type APIKeyRecord struct {
	AccountID string
	APIKeyID  string
	PlanCode  string
	ClientType string
	OverageEnabled bool
}

// LookupAPIKey verifies an API key and returns metadata
func (r *Repository) LookupAPIKey(ctx context.Context, rawKey string) (*APIKeyRecord, error) {
	if r == nil || r.db == nil || !r.db.IsConfigured() {
		return nil, pgx.ErrNoRows
	}
	env, id, secret, ok := ParseAPIKey(rawKey)
	_ = env // future use
	if !ok {
		return nil, errors.New("invalid key format")
	}
	// Query key and subscription
	row := r.db.QueryRow(ctx, `
		SELECT k.id, k.account_id, k.key_hash, k.client_type,
		       COALESCE(s.plan_code, 'lite') AS plan_code,
		       COALESCE(s.overage_enabled, false) AS overage_enabled
		FROM api_keys k
		LEFT JOIN subscriptions s ON s.account_id = k.account_id AND s.status IN ('active','trialing')
		WHERE k.key_prefix = $1 AND k.status = 'active'
	`, id)
	var (
		keyID string
		accountID string
		hash []byte
		clientType string
		planCode string
		overage bool
	)
	scan := row.(interface{ Scan(dest ...any) error })
	if err := scan.Scan(&keyID, &accountID, &hash, &clientType, &planCode, &overage); err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword(hash, []byte(secret)); err != nil {
		return nil, errors.New("invalid api key")
	}
	return &APIKeyRecord{
		AccountID: accountID,
		APIKeyID:  keyID,
		PlanCode:  planCode,
		ClientType: clientType,
		OverageEnabled: overage,
	}, nil
}

// ListAPIKeyIDsByAccount returns key_prefix (id) list for an account
func (r *Repository) ListAPIKeyIDsByAccount(ctx context.Context, accountID string) ([]string, error) {
	if r == nil || r.db == nil || !r.db.IsConfigured() { return nil, errors.New("db not configured") }
	rows, err := r.db.Query(ctx, "SELECT key_prefix FROM api_keys WHERE account_id=$1 AND status='active'", accountID)
	if err != nil { return nil, err }
	type scanner interface{ Next() bool; Scan(dest ...any) error }
	s, ok := rows.(scanner)
	if !ok { return nil, errors.New("invalid rows") }
	var ids []string
	for s.Next() {
		var id string
		if err := s.Scan(&id); err == nil { ids = append(ids, id) }
	}
	return ids, nil
}

// CreateAPIKey inserts a new key and returns the raw key
func (r *Repository) CreateAPIKey(ctx context.Context, accountID string, clientType string, label string, env string) (rawKey string, keyID string, err error) {
	if r == nil || r.db == nil || !r.db.IsConfigured() {
		return "", "", errors.New("db not configured")
	}
	id, raw, hash, err := GenerateAPIKey(env)
	if err != nil {
		return "", "", err
	}
	err = r.db.Exec(ctx, `
		INSERT INTO api_keys(id, account_id, label, key_prefix, key_hash, client_type, status)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, 'active')
	`, accountID, label, id, hash, clientType)
	if err != nil {
		return "", "", err
	}
	return raw, id, nil
}

// RevokeAPIKey marks a key as revoked
func (r *Repository) RevokeAPIKey(ctx context.Context, keyID string) error {
	if r == nil || r.db == nil || !r.db.IsConfigured() {
		return errors.New("db not configured")
	}
return r.db.Exec(ctx, `UPDATE api_keys SET status='revoked' WHERE key_prefix=$1`, keyID)
}

// ListAllActiveAPIKeys returns tuples of (account_id, key_prefix)
func (r *Repository) ListAllActiveAPIKeys(ctx context.Context) ([]struct{ AccountID, KeyID string }, error) {
	if r == nil || r.db == nil || !r.db.IsConfigured() { return nil, errors.New("db not configured") }
	rows, err := r.db.Query(ctx, "SELECT account_id, key_prefix FROM api_keys WHERE status='active'")
	if err != nil { return nil, err }
	type scanner interface{ Next() bool; Scan(dest ...any) error }
	s, ok := rows.(scanner)
	if !ok { return nil, errors.New("invalid rows") }
	var list []struct{ AccountID, KeyID string }
	for s.Next() {
		var a, k string
		if err := s.Scan(&a, &k); err == nil { list = append(list, struct{AccountID, KeyID string}{a,k}) }
	}
	return list, nil
}
