package auth

import (
	"context"
)

// Principal carries authenticated caller metadata derived from the API key
// NOTE: Do not place secrets or raw API keys here.
type Principal struct {
	AccountID      string
	APIKeyID       string
	PlanCode       string // e.g., lite, pro
	ClientType     string // agent or human
	OverageEnabled bool
}

type principalKeyType struct{}

var principalKey = principalKeyType{}

// WithPrincipal attaches principal to context
func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

// GetPrincipal retrieves principal from context (nil if absent)
func GetPrincipal(ctx context.Context) *Principal {
	v := ctx.Value(principalKey)
	if v == nil {
		return nil
	}
	p, _ := v.(*Principal)
	return p
}
