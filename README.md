# SupplyChain API

This repository contains an MVP implementation and design docs for a real-time supply chain disruption alerts service.

Contents:
- Go service: ingestion pipeline, keyword classifier, regex geocoder, and /v1/alerts API
- OpenAPI (openapi.yaml)
- Dockerfile and CI skeleton
- High-level architecture diagrams and roadmap

High-level Architecture (ASCII):

[Data Sources] --(fetchers)--> [Ingestion Pipeline] --(classify+geocode)--> [Store]
   | RSS | Social | Ports |                               | In-memory / Postgres |
                                                   \
                                                    --> [API /v1/alerts]

Components:
- Data ingestion: modular Source interface with an RSS source example. Add social media, port status, and shipping APIs similarly with per-source rate limiters.
- Real-time processing: goroutine per source with ticker; global rate limiter; classification and geocoding stubs.
- Storage: in-memory store for MVP; pluggable Postgres store for production (schema sketched below).
- API: chi-based HTTP server with /v1/alerts and health endpoint.

Database schema (Postgres suggestion):
- alerts(id text PK, source text, title text, summary text, url text unique, detected_at timestamptz, published_at timestamptz, region text, country text, location text, lat double, lon double, disruption text, severity text, sentiment text, confidence double)
Indexes:
- idx_alerts_detected_at DESC
- idx_alerts_country_disruption
- gist index on geography(point) for geo queries

CI/CD:
- Lint, test, build, docker build & push, deploy to Cloud Run/Fargate. See .github/workflows/ci.yaml.

Security:
- Plan for API keys (header: X-API-Key) with tiered quotas and features; OAuth for enterprise webhooks.
- PII: avoid storing user data; location pertains to public events. GDPR: allow deletion/anonymization if user-submitted data ever appears.

Roadmap:
- Replace keyword classifier with spaGO or ONNX runtime model for multi-label classification and sentiment.
- Add geocoding via a managed API with caching (e.g., Pelias/Photon or provider SDK) and normalize to ISO country/region codes.
- Add more sources (Twitter/X, port APIs) with source-specific parsers and backoff.
- Introduce Postgres/TimescaleDB backend, and Redis for dedup + rate limiting.
- Add GraphQL gateway for flexible querying and webhooks for real-time pushes.

Run locally:
- go run ./
- curl 'http://localhost:8080/v1/alerts?limit=10'

