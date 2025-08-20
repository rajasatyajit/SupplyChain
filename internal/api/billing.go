package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rajasatyajit/SupplyChain/internal/auth"
	"github.com/rajasatyajit/SupplyChain/internal/billing"
	stripe "github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/usageRecord"
	"github.com/stripe/stripe-go/v76/webhook"
)

// createCheckoutSession starts a Stripe Checkout session
func (h *Handler) createCheckoutSession(w http.ResponseWriter, r *http.Request) {
	p := auth.GetPrincipal(r.Context())
	if p == nil { h.writeErrorResponse(w, r, http.StatusUnauthorized, "unauthorized"); return }
	var body struct {
		PlanCode string `json:"plan_code"` // lite | pro
		Interval string `json:"interval"` // month | year
		Overage  bool   `json:"overage_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeErrorResponse(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	if body.Interval == "" { body.Interval = "month" }

	bs := billing.NewService(h.db.Config().Billing, h.db)
	url, err := bs.CreateCheckoutSession(r.Context(), p.AccountID, strings.ToLower(body.PlanCode), strings.ToLower(body.Interval), body.Overage)
	if err != nil {
		h.writeErrorResponse(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSONResponse(w, http.StatusOK, map[string]string{"url": url})
}

// createPortalSession creates a Stripe Billing Portal session
func (h *Handler) createPortalSession(w http.ResponseWriter, r *http.Request) {
	p := auth.GetPrincipal(r.Context())
	if p == nil { h.writeErrorResponse(w, r, http.StatusUnauthorized, "unauthorized"); return }
	// Fetch stripe_customer_id for the account
	row := h.db.QueryRow(r.Context(), "SELECT stripe_customer_id FROM subscriptions WHERE account_id=$1 AND status IN ('active','trialing') ORDER BY updated_at DESC LIMIT 1", p.AccountID)
	var custID string
	if err := scanRow(row, &custID); err != nil || custID == "" {
		h.writeErrorResponse(w, r, http.StatusBadRequest, "no active subscription")
		return
	}
	bs := billing.NewService(h.db.Config().Billing, h.db)
	url, err := bs.CreatePortalSession(r.Context(), custID)
	if err != nil {
		h.writeErrorResponse(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSONResponse(w, http.StatusOK, map[string]string{"url": url})
}

func meteredUsageCreate(subscriptionItemID string, quantity int64) (string, error) {
	params := &stripe.UsageRecordParams{
		SubscriptionItem: stripe.String(subscriptionItemID),
		Quantity:         stripe.Int64(quantity),
		Timestamp:        stripe.Int64(time.Now().Unix()),
		Action:           stripe.String(string(stripe.UsageRecordActionIncrement)),
	}
	ur, err := usageRecord.New(params)
	if err != nil { return "", err }
	return ur.ID, nil
}

// stripeWebhook receives Stripe events
func (h *Handler) stripeWebhook(w http.ResponseWriter, r *http.Request) {
	payload, _ := ioutil.ReadAll(r.Body)
	sig := r.Header.Get("Stripe-Signature")
	secret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	event, err := webhook.ConstructEvent(payload, sig, secret)
	if err != nil {
		h.writeErrorResponse(w, r, http.StatusBadRequest, "invalid signature")
		return
	}
	switch event.Type {
	case "checkout.session.completed":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err == nil {
			accountID := ""
			if sess.Subscription != nil {
				// Store customer and subscription IDs
				h.db.Exec(r.Context(), "UPDATE subscriptions SET stripe_customer_id=$1, stripe_subscription_id=$2, status='active', updated_at=now() WHERE account_id=$3", sess.Customer.ID, sess.Subscription.ID, sess.ClientReferenceID)
			} else {
				accountID = sess.ClientReferenceID
				// Create subscription row if missing
				h.db.Exec(r.Context(), "INSERT INTO subscriptions(account_id, plan_code, status, created_at, updated_at) VALUES ($1, $2, 'trialing', now(), now()) ON CONFLICT (account_id) DO NOTHING", accountID, sess.Metadata["plan_code"])
			}
		}
	case "customer.subscription.created", "customer.subscription.updated":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err == nil {
			planCode := sub.Metadata["plan_code"]
			over := sub.Metadata["overage_enabled"] == "true"
			periodStart := sub.CurrentPeriodStart
			periodEnd := sub.CurrentPeriodEnd
			status := string(sub.Status)
			// We need the account_id from metadata if present
			acct := sub.Metadata["account_id"]
			h.db.Exec(r.Context(), "UPDATE subscriptions SET plan_code=$1, overage_enabled=$2, status=$3, current_period_start=to_timestamp($4), current_period_end=to_timestamp($5), updated_at=now() WHERE stripe_subscription_id=$6 OR account_id=$7", planCode, over, status, periodStart, periodEnd, sub.ID, acct)
		}
case "invoice.finalized":
		// On invoice finalization, report overage usage if enabled
		var inv stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &inv); err == nil {
			// Find the subscription id and items
			subID := ""
			if inv.Subscription != nil { subID = inv.Subscription.ID }
			if subID != "" {
				// Find account_id and plan_code from subscription metadata if present
				// We need account_id to sum quotas across keys
				// Fetch account_id from DB via subscription id
				row := h.db.QueryRow(r.Context(), "SELECT account_id, plan_code, overage_enabled FROM subscriptions WHERE stripe_subscription_id=$1", subID)
				var acctID, planCode string; var over bool
				_ = scanRow(row, &acctID, &planCode, &over)
				if over {
					// Sum usage for all keys of this account from Redis and compute overage
					ids, _ := auth.NewRepository(h.db).ListAPIKeyIDsByAccount(r.Context(), acctID)
					mgr := getRateLimiter()
					if mgr != nil {
						now := time.Now().UTC()
						total, _ := mgr.SumQuotas(r.Context(), ids, now)
						// plan monthly
						rpm, monthly := mgr.PlanLimits(planCode)
						_ = rpm
						overUnits := total - monthly
						if overUnits > 0 {
							// find metered item price on the subscription
							var meteredItemID string
							for _, li := range inv.Lines.Data {
								if li.Price != nil && li.Price.ID == h.db.Config().Billing.PriceOverageMetered {
									meteredItemID = li.SubscriptionItem
									break
								}
							}
							if meteredItemID != "" {
								// Report usage as one record at period end
								// For simplicity, report now; Stripe will attribute to current period
								_, _ = meteredUsageCreate(meteredItemID, int64(overUnits))
							}
						}
					}
				}
			}
		}
	case "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err == nil {
			h.db.Exec(r.Context(), "UPDATE subscriptions SET status='canceled', updated_at=now() WHERE stripe_subscription_id=$1", sub.ID)
		}
	}
	w.WriteHeader(http.StatusOK)
}
