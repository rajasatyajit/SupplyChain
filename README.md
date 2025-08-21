# SupplyChain Alert System


A production-ready Go application for monitoring and analyzing supply chain disruptions through real-time alert processing.

## Features

- **Real-time Alert Processing**: Fetches and processes alerts from multiple sources (RSS feeds, APIs)
- **Intelligent Classification**: Automatically categorizes alerts by severity, sentiment, and disruption type
- **Geolocation**: Extracts and geocodes location information from alert content
- **RESTful API**: Comprehensive HTTP API with health checks and metrics
- **Production Ready**: Structured logging, metrics, graceful shutdown, and monitoring
- **Scalable Architecture**: Configurable pipeline with rate limiting and concurrent processing
- **Database Support**: PostgreSQL with connection pooling and migrations
- **Containerized**: Docker support with multi-stage builds and security best practices

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/rajasatyajit/SupplyChain.git
cd SupplyChain

# Start all services
docker-compose up -d

# Check application health
curl http://localhost:8080/health

# View logs
docker-compose logs -f app
```

### Local Development

```bash
# Install dependencies
go mod download

# Set up environment
cp .env.example .env
# Edit .env with your configuration

# Run database migrations (if using PostgreSQL)
psql -h localhost -U supplychain -d supplychain -f scripts/init.sql

# Build and run
make build
./build/supplychain

# Or run directly
make run
```

## API Endpoints

- Detailed reference: see docs/API.md

### Health Checks
- `GET /health` - Basic health check
- `GET /v1/health/ready` - Readiness check (includes database)
- `GET /v1/health/live` - Liveness check

### Alerts
- `GET /v1/alerts` - List alerts with filtering
- `GET /v1/alerts/{id}` - Get specific alert

### System
- `GET /v1/version` - Application version info
- `GET /metrics` - Prometheus metrics (port 9090)

### Query Parameters for `/v1/alerts`

```
?source=rss&severity=high&limit=100&since=2024-01-01T00:00:00Z
```

Available filters:
- `source` - Filter by alert source
- `severity` - Filter by severity (low, medium, high)
- `disruption` - Filter by disruption type
- `region` - Filter by region
- `country` - Filter by country
- `since` - Filter alerts after timestamp
- `until` - Filter alerts before timestamp
- `limit` - Limit number of results (max 1000)

## Configuration

Configuration is handled through environment variables. See `.env.example` for all available options.

### Key Configuration Options

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | 8080 | HTTP server port |
| `DATABASE_URL` | - | PostgreSQL connection string |
| `LOG_LEVEL` | info | Logging level (debug, info, warn, error) |
| `LOG_FORMAT` | json | Log format (json, text) |
| `PIPELINE_RATE_LIMIT` | 5.0 | Requests per second limit |
| `PIPELINE_WORKER_COUNT` | 4 | Number of concurrent workers |
| `METRICS_ENABLED` | true | Enable Prometheus metrics |

## Development

### Prerequisites

- Go 1.22+
- PostgreSQL 13+ (optional, falls back to in-memory storage)
- Docker & Docker Compose (for containerized development)

### Make Targets

```bash
make help           # Show all available targets
make build          # Build the application
make test           # Run tests
make test-coverage  # Run tests with coverage
make lint           # Run linting
make fmt            # Format code
make clean          # Clean build artifacts
make docker-build   # Build Docker image
make ci             # Run full CI pipeline
make dev            # Development setup
```

### Project Structure

```
├── cmd/supplychain/     # Application entry point
├── config/              # Configuration management
├── internal/            # Internal packages
│   ├── api/            # HTTP API handlers
│   ├── classifier/     # Alert classification
│   ├── database/       # Database layer
│   ├── geocoder/       # Geolocation services
│   ├── logger/         # Structured logging
│   ├── metrics/        # Metrics collection
│   ├── middleware/     # HTTP middleware
│   ├── models/         # Data models
│   ├── pipeline/       # Data processing pipeline
│   └── store/          # Data persistence
├── pkg/utils/          # Utility functions
├── deployments/        # Kubernetes manifests
├── docs/               # Documentation
├── monitoring/         # Monitoring configuration
├── scripts/            # Database scripts
├── test/               # Test files
├── Dockerfile          # Container definition
├── docker-compose.yml  # Development environment
└── Makefile           # Build automation
```

## Monitoring

The application includes comprehensive monitoring:

### Metrics (Prometheus)
- HTTP request metrics (duration, status codes)
- Pipeline processing metrics
- Database connection metrics
- Custom business metrics

Access metrics at: `http://localhost:9090/metrics`

### Logging (Structured JSON)
- Request/response logging
- Error tracking
- Performance monitoring
- Audit trails

### Health Checks
- Kubernetes-ready health endpoints
- Database connectivity checks
- Dependency health monitoring

## Deployment

### Docker

```bash
# Build production image
make docker-build

# Run container
docker run -p 8080:8080 -e DATABASE_URL="..." supplychain:latest
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: supplychain
spec:
  replicas: 3
  selector:
    matchLabels:
      app: supplychain
  template:
    metadata:
      labels:
        app: supplychain
    spec:
      containers:
      - name: supplychain
        image: supplychain:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: supplychain-secrets
              key: database-url
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

## Security

- Non-root container execution
- Security headers middleware
- Input validation and sanitization
- Rate limiting
- Structured error handling (no sensitive data exposure)

## Performance

- Connection pooling for database
- Concurrent pipeline processing
- Rate limiting and backpressure
- Efficient JSON serialization
- Proper resource cleanup

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow Go best practices and idioms
- Write tests for new functionality
- Update documentation for API changes
- Use structured logging
- Add metrics for new features

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

For support and questions:
- Create an issue in the GitHub repository
- Check the documentation and examples
- Review the logs for troubleshooting information