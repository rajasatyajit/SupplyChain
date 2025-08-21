# SupplyChain API Documentation

## Overview

This document complements the README and focuses on auth, billing, rate limits, and example requests/responses.

The SupplyChain API provides access to supply chain disruption alerts and monitoring data. The API follows REST principles and returns JSON responses.

## Base URL

```
https://api.supplychain.example.com/v1
```

## Authentication

Use API keys via the Authorization header. Keys can be created and managed in your dashboard.

Headers:
- Authorization: Bearer <API_KEY>
- X-Client-Type: agent or human (recommended; helps us tailor limits and guidance)

Trials: New accounts receive 10 API calls total across endpoints before requiring a paid plan.

## Rate Limiting

The service enforces per-endpoint RPM and per-account monthly quotas using Redis when configured, with in-memory fallback. Trial accounts without an active/trialing subscription are limited to 10 requests total across endpoints.

Response headers include:
- X-RateLimit-Limit, X-RateLimit-Reset, X-RateLimit-Remaining
- X-Quota-Limit, X-Quota-Remaining, X-Quota-Reset
- Retry-After on 429

Rate limits are per API key and per endpoint. Plans define both per-minute burst and monthly quotas.
- Lite: 20 requests/min per endpoint, up to 450,000 requests/month per API key
- Pro: 60 requests/min per endpoint, up to 1,350,000 requests/month per API key
- Overage: Optional. If enabled, requests beyond monthly quota are billed at $0.000033 per request.

On limit exceed, you may receive 429 Too Many Requests. Responses include headers:
- X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset
- X-Quota-Limit, X-Quota-Remaining, X-Quota-Reset
- Retry-After (on 429)

## Health Checks

### GET /health
Basic health check endpoint.

**Response:**
```json
{
  "status": "ok",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0"
}
```

### GET /v1/health/ready
Readiness check that includes dependency health.

**Response:**
```json
{
  "status": "ready",
  "timestamp": "2024-01-15T10:30:00Z",
  "checks": {
    "store": "ok",
    "database": "ok"
  }
}
```

### GET /v1/health/live
Liveness check for container orchestration.

**Response:**
```json
{
  "status": "alive",
  "timestamp": "2024-01-15T10:30:00Z",
  "uptime": "2h30m15s"
}
```

## Alerts

### GET /v1/alerts
Retrieve alerts with optional filtering.

**Query Parameters:**
- `source` - Filter by alert source
- `severity` - Filter by severity (low, medium, high)
- `disruption` - Filter by disruption type
- `region` - Filter by geographical region
- `country` - Filter by country
- `since` - Filter alerts after timestamp (RFC3339 format)
- `until` - Filter alerts before timestamp (RFC3339 format)
- `limit` - Limit number of results (max 1000, default 100)
- `offset` - Offset for pagination

**Example Request:**
```
GET /v1/alerts?severity=high&limit=50&since=2024-01-01T00:00:00Z
```

**Response:**
```json
{
  "data": [
    {
      "id": "alert-123",
      "source": "Global Shipping News",
      "title": "Port Strike Disrupts West Coast Operations",
      "summary": "Major port strike affecting container operations...",
      "url": "https://example.com/news/port-strike",
      "detected_at": "2024-01-15T10:30:00Z",
      "published_at": "2024-01-15T09:00:00Z",
      "region": "North America",
      "country": "United States",
      "location": "Port of Los Angeles",
      "latitude": 33.7361,
      "longitude": -118.2922,
      "disruption": "port_status",
      "severity": "high",
      "sentiment": "negative",
      "confidence": 0.92,
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ],
  "count": 1,
  "timestamp": "2024-01-15T10:35:00Z"
}
```

### GET /v1/alerts/{id}
Retrieve a specific alert by ID.

**Response:**
```json
{
  "id": "alert-123",
  "source": "Global Shipping News",
  "title": "Port Strike Disrupts West Coast Operations",
  "summary": "Major port strike affecting container operations...",
  "url": "https://example.com/news/port-strike",
  "detected_at": "2024-01-15T10:30:00Z",
  "published_at": "2024-01-15T09:00:00Z",
  "region": "North America",
  "country": "United States",
  "location": "Port of Los Angeles",
  "latitude": 33.7361,
  "longitude": -118.2922,
  "disruption": "port_status",
  "severity": "high",
  "sentiment": "negative",
  "confidence": 0.92,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

## Billing

### Checkout

Stripe (default):
- POST /v1/billing/checkout-session
- Body: {"plan_code":"lite|pro","interval":"month|year","overage_enabled":true|false}
- Response: {"provider":"stripe","url":"https://checkout.stripe.com/..."}

Razorpay (India or ?provider=razorpay):
- POST /v1/billing/checkout-session?provider=razorpay
- Body: {"plan_code":"lite|pro","interval":"month|year"}
- Response: {"provider":"razorpay","params": {"key":"...","order_id":"order_...","amount":49900,"currency":"INR","notes":{...}}}

### Portal
- POST /v1/billing/portal-session (Stripe only)
- Response: {"url":"..."}

### Webhooks
Stripe:
- POST /v1/billing/webhook
- Header: Stripe-Signature, secret=STRIPE_WEBHOOK_SECRET
- Events: checkout.session.completed, customer.subscription.created|updated, invoice.finalized (reports overage usage), customer.subscription.deleted

Razorpay:
- POST /v1/billing/razorpay/webhook
- Header: X-Razorpay-Signature (HMAC SHA256 of raw body)
- Events handled: payment.captured, order.paid (activates subscription using notes.account_id, notes.plan_code)

### Examples
Stripe checkout request:
```
POST /v1/billing/checkout-session
Authorization: Bearer sk_test
Content-Type: application/json

{"plan_code":"lite","interval":"month","overage_enabled":true}
```
Stripe checkout response:
```
{"provider":"stripe","url":"https://checkout.stripe.com/c/session/abc"}
```

Razorpay checkout request:
```
POST /v1/billing/checkout-session?provider=razorpay
Authorization: Bearer sk_test_in
Content-Type: application/json

{"plan_code":"lite","interval":"month"}
```
Razorpay checkout response:
```
{
  "provider":"razorpay",
  "params":{
    "key":"rzp_test_123",
    "order_id":"order_abc",
    "amount":49900,
    "currency":"INR",
    "notes":{"account_id":"acc_123","plan_code":"lite","interval":"month"}
  }
}
```

Razorpay webhook example (order.paid):
```
{
  "event":"order.paid",
  "payload":{
    "order":{
      "entity":{
        "id":"order_abc",
        "notes":{"account_id":"acc_123","plan_code":"lite"}
      }
    }
  }
}
```

## System Information

### GET /v1/version
Get application version information.

**Response:**
```json
{
  "version": "1.0.0",
  "build_time": "2024-01-15T08:00:00Z",
  "git_commit": "abc123def"
}
```

## Error Responses

All error responses follow a consistent format:

```json
{
  "error": "Bad Request",
  "message": "Invalid limit parameter",
  "timestamp": "2024-01-15T10:30:00Z",
  "request_id": "req-123-456"
}
```

### HTTP Status Codes

- `200` - Success
- `400` - Bad Request (invalid parameters)
- `404` - Not Found
- `429` - Too Many Requests (rate limited)
- `500` - Internal Server Error

## Data Models

### Alert

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique alert identifier |
| source | string | Data source name |
| title | string | Alert title |
| summary | string | Alert description |
| url | string | Source URL |
| detected_at | timestamp | When alert was detected |
| published_at | timestamp | When alert was originally published |
| region | string | Geographical region |
| country | string | Country |
| location | string | Specific location |
| latitude | number | Latitude coordinate |
| longitude | number | Longitude coordinate |
| disruption | string | Type of disruption (port_status, rail, road, air, general) |
| severity | string | Severity level (low, medium, high) |
| sentiment | string | Sentiment analysis (positive, neutral, negative) |
| confidence | number | Confidence score (0.0 - 1.0) |
| created_at | timestamp | Record creation time |
| updated_at | timestamp | Record last update time |

## Examples

### Get High Severity Alerts from Last 24 Hours

```bash
curl "https://api.supplychain.example.com/v1/alerts?severity=high&since=$(date -u -d '24 hours ago' +%Y-%m-%dT%H:%M:%SZ)"
```

### Get Port-Related Alerts

```bash
curl "https://api.supplychain.example.com/v1/alerts?disruption=port_status&limit=20"
```

### Get Alerts for Specific Region

```bash
curl "https://api.supplychain.example.com/v1/alerts?region=North%20America&country=United%20States"
```