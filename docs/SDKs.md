# Official SDKs (Minimal)

These SDKs provide a thin wrapper over the HTTP API. Set API_KEY and X-Client-Type (agent|human).

- Go: see sdk/go
- Python: see sdk/python
- JavaScript (Node): see sdk/js

All SDKs expose methods:
- Alerts(params)
- Usage()
- UsageTimeseries(bucket, start, end)
- CreateCheckoutSession(planCode, interval, overageEnabled)
- CreatePortalSession()
