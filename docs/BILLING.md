# Billing and Payments

- Stripe Checkout + Billing handles subscriptions (monthly and annual) and the Billing Portal (optional) for plan management.
- Prices are displayed in USD. Settlement into your Indian bank account occurs in INR; Stripe performs FX conversion.
- Overage charges: $0.000033 per request beyond included monthly quota when overage is enabled on the account.
- Refunds: refund = amount paid - prorated usage cost to date. Manual action via owner dashboard initially.
- Webhooks: we process Stripe events to keep subscription state in sync and to finalize usage on invoice.

Endpoints:
- POST /v1/billing/checkout-session
  Body: { "plan_code": "lite|pro", "interval": "month|year", "overage_enabled": true|false }
  Returns: { "url": "https://checkout.stripe.com/..." }
- POST /v1/billing/portal-session
  Returns: { "url": "https://billing.stripe.com/..." }
- POST /v1/billing/webhook (Stripe)

Setup steps:
1. Create Stripe products and prices for Lite/Pro (monthly and annual) and a metered overage price at $0.000033/request.
2. Put price IDs into env:
   - STRIPE_PRICE_LITE_MONTHLY, STRIPE_PRICE_LITE_ANNUAL
   - STRIPE_PRICE_PRO_MONTHLY, STRIPE_PRICE_PRO_ANNUAL
   - STRIPE_PRICE_OVERAGE_METERED
3. Configure webhook to POST to https://api.yourdomain.com/v1/billing/webhook and set STRIPE_WEBHOOK_SECRET.
4. Set STRIPE_CHECKOUT_SUCCESS_URL, STRIPE_CHECKOUT_CANCEL_URL, STRIPE_PORTAL_RETURN_URL to your dashboard URLs.

Notes:
- On invoice.finalized, we compute overage units based on Redis usage totals and report them to the metered item.
- Trial: capped at 10 total calls until subscription is trialing/active.

Razorpay (optional, India domestic)
- Configure RAZORPAY_KEY_ID, RAZORPAY_KEY_SECRET, RAZORPAY_WEBHOOK_SECRET.
- To route checkout to Razorpay, pass provider=razorpay in query OR set header X-Country: IN.
- Webhook: POST /v1/billing/razorpay/webhook (signature verification to be completed; scaffold in place).
- Typical flow: server creates order (or returns checkout parameters), front-end completes payment via hosted checkout, webhook confirms and maps to local subscription.
