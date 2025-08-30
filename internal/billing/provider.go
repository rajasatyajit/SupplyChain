package billing

import (
	"context"
	"net/http"

	"github.com/rajasatyajit/SupplyChain/internal/database"
)

// CheckoutResponse is a generic response for initiating checkout across providers
// For Stripe, URL is populated. For Razorpay, Params contains order create payload for frontend.
type CheckoutResponse struct {
	Provider string                 `json:"provider"`
	URL      string                 `json:"url,omitempty"`
	Params   map[string]interface{} `json:"params,omitempty"`
}

type Provider interface {
	Name() string
	CreateCheckout(ctx context.Context, accountID, planCode, interval string, overage bool) (CheckoutResponse, error)
	CreatePortal(ctx context.Context, customerID string) (string, error)
	VerifyWebhook(r *http.Request, body []byte) error
	HandleWebhook(ctx context.Context, db *database.DB, body []byte) error
}
