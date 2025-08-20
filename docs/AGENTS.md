# AI Agents Integration Guide

This guide explains best practices for AI agents interacting with the SupplyChain API.

Key points for agents:
- Send headers:
  - Authorization: Bearer <API_KEY>
  - X-Client-Type: agent
- Respect rate-limit and quota headers:
  - X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset
  - X-Quota-Limit, X-Quota-Remaining, X-Quota-Reset
- On 429, back off using Retry-After and jitter. Exponential backoff recommended.
- Discover current limits via GET /v1/limits and adapt concurrency accordingly.
- Monitor usage via GET /v1/usage and reduce load as you approach quotas.

Curl example:

```bash
curl -H "Authorization: Bearer $API_KEY" \
     -H "X-Client-Type: agent" \
     "https://api.supplychain.example.com/v1/alerts?limit=10"
```
