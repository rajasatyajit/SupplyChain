package billing

import (
	"context"
	"net/http"

	"github.com/rajasatyajit/SupplyChain/internal/database"
)

type StripeProvider struct{ svc *Service }

func NewStripeProvider(svc *Service) *StripeProvider { return &StripeProvider{svc: svc} }

func (p *StripeProvider) Name() string { return "stripe" }

func (p *StripeProvider) CreateCheckout(ctx context.Context, accountID, planCode, interval string, overage bool) (CheckoutResponse, error) {
	url, err := p.svc.CreateCheckoutSession(ctx, accountID, planCode, interval, overage)
	if err != nil {
		return CheckoutResponse{}, err
	}
	return CheckoutResponse{Provider: p.Name(), URL: url}, nil
}

func (p *StripeProvider) CreatePortal(ctx context.Context, customerID string) (string, error) {
	return p.svc.CreatePortalSession(ctx, customerID)
}

// Stripe webhook verification is handled in api via stripe-go; so this is a no-op here.
func (p *StripeProvider) VerifyWebhook(r *http.Request, body []byte) error { return nil }

// For now, Stripe webhook handling remains in api handler. Implementing here later would centralize logic.
func (p *StripeProvider) HandleWebhook(ctx context.Context, db *database.DB, body []byte) error {
	return nil
}
