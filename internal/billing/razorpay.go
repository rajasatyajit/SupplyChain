package billing

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rajasatyajit/SupplyChain/config"
	"github.com/rajasatyajit/SupplyChain/internal/database"
)

// Razorpay service using Orders API
// For subscriptions, consider Razorpay Subscriptions API. Here we use Orders for initial checkout.
type RazorpayService struct {
	cfg    config.BillingConfig
	client *http.Client
}

func NewRazorpay(cfg config.BillingConfig) *RazorpayService {
	return &RazorpayService{cfg: cfg, client: &http.Client{Timeout: 10 * time.Second}}
}

func (r *RazorpayService) Name() string { return "razorpay" }

// CreateCheckout creates an order and returns parameters needed by frontend checkout
func (r *RazorpayService) CreateCheckout(ctx context.Context, accountID string, planCode string, interval string, _ bool) (CheckoutResponse, error) {
	if r.cfg.RazorpayKeyID == "" || r.cfg.RazorpayKeySecret == "" {
		return CheckoutResponse{}, errors.New("razorpay not configured")
	}
	amount, currency, err := r.amountForPlan(planCode, interval)
	if err != nil {
		return CheckoutResponse{}, err
	}

	order, err := r.createOrder(ctx, amount, currency, map[string]string{
		"account_id": accountID,
		"plan_code":  planCode,
		"interval":   interval,
	})
	if err != nil {
		return CheckoutResponse{}, err
	}
	params := map[string]interface{}{
		"key":        r.cfg.RazorpayKeyID,
		"amount":     order.Amount,
		"currency":   order.Currency,
		"order_id":   order.ID,
		"notes":      order.Notes,
		"account_id": accountID,
	}
	return CheckoutResponse{Provider: r.Name(), Params: params}, nil
}

func (r *RazorpayService) CreatePortal(ctx context.Context, customerID string) (string, error) {
	return "", errors.New("razorpay portal not supported")
}

func (r *RazorpayService) VerifyWebhook(req *http.Request, body []byte) error {
	sig := req.Header.Get("X-Razorpay-Signature")
	if sig == "" {
		return errors.New("missing signature")
	}
	mac := hmac.New(sha256.New, []byte(r.cfg.RazorpayWebhookSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	// Signature may be base64 or hex depending on configuration; handle common case hex
	if hmac.Equal([]byte(expected), []byte(sig)) {
		return nil
	}
	// Try base64 comparison as fallback
	if dec, err := base64.StdEncoding.DecodeString(sig); err == nil {
		if hmac.Equal(mac.Sum(nil), dec) {
			return nil
		}
	}
	return errors.New("invalid signature")
}

func (r *RazorpayService) HandleWebhook(ctx context.Context, db *database.DB, body []byte) error {
	// Parse event
	var evt struct {
		Event   string         `json:"event"`
		Payload map[string]any `json:"payload"`
	}
	if err := json.Unmarshal(body, &evt); err != nil {
		return err
	}
	// We care about payment.captured and order.paid
	switch evt.Event {
	case "payment.captured":
		// payload.payment.entity has order_id and notes
		p, _ := evt.Payload["payment"].(map[string]any)
		ent, _ := p["entity"].(map[string]any)
		notes, _ := ent["notes"].(map[string]any)
		accountID, _ := notes["account_id"].(string)
		planCode, _ := notes["plan_code"].(string)
		if accountID != "" {
			// Upsert subscription row as active
			_ = db.Exec(ctx, "INSERT INTO subscriptions(account_id, plan_code, status, updated_at) VALUES ($1,$2,'active', now()) ON CONFLICT (account_id) DO UPDATE SET plan_code=excluded.plan_code, status='active', updated_at=now()", accountID, planCode)
		}
	case "order.paid":
		// payload.order.entity has notes we set during order creation
		o, _ := evt.Payload["order"].(map[string]any)
		ent, _ := o["entity"].(map[string]any)
		notes, _ := ent["notes"].(map[string]any)
		accountID, _ := notes["account_id"].(string)
		planCode, _ := notes["plan_code"].(string)
		if accountID != "" {
			_ = db.Exec(ctx, "INSERT INTO subscriptions(account_id, plan_code, status, updated_at) VALUES ($1,$2,'active', now()) ON CONFLICT (account_id) DO UPDATE SET plan_code=excluded.plan_code, status='active', updated_at=now()", accountID, planCode)
		}
	default:
		// ignore other events for now
	}
	return nil
}

// ParseWebhook extracts event type and payload metadata
func (r *RazorpayService) ParseWebhook(body []byte) (string, map[string]interface{}, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal(body, &obj); err != nil {
		return "", nil, err
	}
	typ, _ := obj["event"].(string)
	return typ, obj, nil
}

// Helper: determine amount based on plan
func (r *RazorpayService) amountForPlan(planCode, interval string) (int64, string, error) {
	cur := r.cfg.RazorpayCurrency
	if cur == "" {
		cur = "INR"
	}
	switch planCode {
	case "lite":
		if interval == "year" {
			return r.cfg.RazorpayAmountLiteAnnualPaisa, cur, nil
		}
		return r.cfg.RazorpayAmountLiteMonthlyPaisa, cur, nil
	case "pro":
		if interval == "year" {
			return r.cfg.RazorpayAmountProAnnualPaisa, cur, nil
		}
		return r.cfg.RazorpayAmountProMonthlyPaisa, cur, nil
	default:
		return 0, cur, errors.New("invalid plan_code")
	}
}

type razorpayOrderRequest struct {
	Amount   int64             `json:"amount"`
	Currency string            `json:"currency"`
	Receipt  string            `json:"receipt,omitempty"`
	Notes    map[string]string `json:"notes,omitempty"`
}

type razorpayOrderResponse struct {
	ID       string            `json:"id"`
	Amount   int64             `json:"amount"`
	Currency string            `json:"currency"`
	Status   string            `json:"status"`
	Notes    map[string]string `json:"notes"`
}

func (r *RazorpayService) createOrder(ctx context.Context, amount int64, currency string, notes map[string]string) (*razorpayOrderResponse, error) {
	payload := razorpayOrderRequest{Amount: amount, Currency: currency, Notes: notes}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.razorpay.com/v1/orders", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(r.cfg.RazorpayKeyID, r.cfg.RazorpayKeySecret)
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("razorpay order failed: %d %s", resp.StatusCode, string(body))
	}
	var out razorpayOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
