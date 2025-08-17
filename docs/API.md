# SupplyChain API Documentation

## Overview

The SupplyChain API provides access to supply chain disruption alerts and monitoring data. The API follows REST principles and returns JSON responses.

## Base URL

```
https://api.supplychain.example.com/v1
```

## Authentication

Currently, the API does not require authentication. In production, you would implement API keys, OAuth, or JWT tokens.

## Rate Limiting

The API implements rate limiting to ensure fair usage:
- 100 requests per minute per IP address
- Rate limit headers are included in responses

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