package billing

import (
	"context"
	"errors"
	"fmt"

	"github.com/rajasatyajit/SupplyChain/config"
	"github.com/rajasatyajit/SupplyChain/internal/database"
	stripe "github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	portal "github.com/stripe/stripe-go/v76/billingportal/session"
)

type Service struct {
	cfg config.BillingConfig
	db  *database.DB
}

func NewService(cfg config.BillingConfig, db *database.DB) *Service {
	stripe.Key = cfg.StripeSecretKey
	return &Service{cfg: cfg, db: db}
}

func (s *Service) CreateCheckoutSession(ctx context.Context, accountID string, planCode string, interval string, overage bool) (string, error) {
	price := ""
	switch planCode {
	case "lite":
		if interval == "year" { price = s.cfg.PriceLiteAnnual } else { price = s.cfg.PriceLiteMonthly }
	case "pro":
		if interval == "year" { price = s.cfg.PriceProAnnual } else { price = s.cfg.PriceProMonthly }
	default:
		return "", errors.New("invalid plan_code")
	}
	if price == "" {
		return "", errors.New("price not configured")
	}
	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(s.cfg.CheckoutSuccessURL),
		CancelURL:  stripe.String(s.cfg.CheckoutCancelURL),
		ClientReferenceID: stripe.String(accountID),
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				"account_id": accountID,
				"plan_code": planCode,
				"overage_enabled": fmt.Sprintf("%t", overage),
			},
		},
	}
	params.LineItems = []*stripe.CheckoutSessionLineItemParams{
		{Price: stripe.String(price), Quantity: stripe.Int64(1)},
	}
	// Optional overage metered item will be attached post-webhook if overage enabled
	sess, err := session.New(params)
	if err != nil { return "", err }
	return sess.URL, nil
}

func (s *Service) CreatePortalSession(ctx context.Context, stripeCustomerID string) (string, error) {
	if stripeCustomerID == "" { return "", errors.New("missing stripe_customer_id") }
	ps, err := portal.New(&stripe.BillingPortalSessionParams{
		Customer:  stripe.String(stripeCustomerID),
		ReturnURL: stripe.String(s.cfg.PortalReturnURL),
	})
	if err != nil { return "", err }
	return ps.URL, nil
}
