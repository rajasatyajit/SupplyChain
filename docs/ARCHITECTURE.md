# SupplyChain Architecture

## Overview

The SupplyChain application is built using a clean architecture pattern with clear separation of concerns, dependency injection, and production-ready practices.

## Project Structure

```
├── cmd/                          # Application entry points
│   └── supplychain/             # Main application
│       └── main.go              # Application bootstrap
├── config/                      # Configuration management
│   └── config.go               # Environment-based configuration
├── deployments/                 # Deployment configurations
│   └── kubernetes/             # Kubernetes manifests
│       ├── deployment.yaml     # Application deployment
│       └── ingress.yaml        # Ingress configuration
├── docs/                       # Documentation
│   ├── API.md                  # API documentation
│   └── ARCHITECTURE.md         # This file
├── internal/                   # Private application code
│   ├── api/                    # HTTP API handlers
│   │   └── handler.go          # REST API implementation
│   ├── classifier/             # Alert classification
│   │   └── classifier.go       # ML/AI classification logic
│   ├── database/               # Database layer
│   │   └── database.go         # PostgreSQL connection management
│   ├── errors/                 # Error definitions
│   │   └── errors.go           # Custom error types
│   ├── geocoder/               # Geolocation services
│   │   └── geocoder.go         # Location extraction and geocoding
│   ├── logger/                 # Structured logging
│   │   └── logger.go           # Centralized logging configuration
│   ├── metrics/                # Observability
│   │   └── metrics.go          # Prometheus metrics
│   ├── middleware/             # HTTP middleware
│   │   └── middleware.go       # Security, logging, metrics middleware
│   ├── models/                 # Data models
│   │   └── alert.go            # Alert data structures
│   ├── pipeline/               # Data processing pipeline
│   │   ├── pipeline.go         # Pipeline orchestration
│   │   └── rss_source.go       # RSS feed data source
│   └── store/                  # Data persistence
│       ├── store.go            # Storage interface
│       ├── postgres.go         # PostgreSQL implementation
│       └── memory.go           # In-memory implementation
├── monitoring/                 # Monitoring configuration
│   └── prometheus.yml          # Prometheus configuration
├── pkg/                        # Public library code
│   └── utils/                  # Utility functions
│       ├── hash.go             # Hashing utilities
│       └── strings.go          # String processing utilities
├── scripts/                    # Database and utility scripts
│   └── init.sql               # Database initialization
├── test/                       # Test files
│   └── integration/           # Integration tests
│       └── api_test.go        # API integration tests
├── docker-compose.yml          # Development environment
├── Dockerfile                  # Container definition
├── Makefile                   # Build automation
└── README.md                  # Project documentation
```

## Architecture Layers

### 1. Presentation Layer (`internal/api`, `internal/middleware`)
- **Responsibility**: HTTP request handling, response formatting, middleware
- **Components**:
  - REST API handlers with proper error handling
  - Security middleware (headers, rate limiting)
  - Logging and metrics middleware
  - Request validation and response formatting

### 2. Business Logic Layer (`internal/classifier`, `internal/geocoder`, `internal/pipeline`)
- **Responsibility**: Core business logic, data processing, AI/ML operations
- **Components**:
  - Alert classification using keyword analysis
  - Geolocation extraction and geocoding
  - Pipeline orchestration for data processing
  - Source management for different data feeds

### 3. Data Layer (`internal/store`, `internal/database`)
- **Responsibility**: Data persistence, database operations
- **Components**:
  - Abstract storage interface for dependency injection
  - PostgreSQL implementation with connection pooling
  - In-memory fallback for development/testing
  - Database health checks and monitoring

### 4. Infrastructure Layer (`config`, `internal/logger`, `internal/metrics`)
- **Responsibility**: Cross-cutting concerns, configuration, observability
- **Components**:
  - Environment-based configuration management
  - Structured JSON logging with context
  - Prometheus metrics collection
  - Error handling and monitoring

## Design Patterns

### Dependency Injection
- Interfaces define contracts between layers
- Concrete implementations are injected at runtime
- Enables easy testing and swapping of components

### Repository Pattern
- `Store` interface abstracts data persistence
- Multiple implementations (PostgreSQL, in-memory)
- Clean separation between business logic and data access

### Pipeline Pattern
- Modular data processing with pluggable sources
- Rate limiting and concurrent processing
- Error handling and retry logic

### Middleware Pattern
- Composable HTTP middleware for cross-cutting concerns
- Security, logging, metrics, and error handling
- Clean separation of concerns

## Data Flow

```
External Sources → Pipeline → Classifier → Geocoder → Store → API → Client
```

1. **Data Ingestion**: Pipeline fetches data from external sources (RSS feeds, APIs)
2. **Processing**: Alerts are classified for severity and sentiment
3. **Enhancement**: Geographic information is extracted and geocoded
4. **Storage**: Processed alerts are stored in the database
5. **API**: REST API provides access to stored alerts with filtering
6. **Monitoring**: All operations are logged and monitored

## Configuration Management

- Environment-based configuration with validation
- Sensible defaults for all settings
- Support for development, staging, and production environments
- Configuration hot-reloading where appropriate

## Security Features

- Non-root container execution
- Security headers middleware
- Input validation and sanitization
- Rate limiting to prevent abuse
- Structured error responses (no sensitive data leakage)

## Observability

### Logging
- Structured JSON logging for production
- Contextual logging with request IDs
- Configurable log levels
- Performance and error tracking

### Metrics
- Prometheus metrics for all components
- HTTP request metrics (duration, status codes)
- Pipeline processing metrics
- Database connection monitoring
- Custom business metrics

### Health Checks
- Kubernetes-ready health endpoints
- Database connectivity checks
- Dependency health monitoring
- Graceful degradation

## Scalability Considerations

### Horizontal Scaling
- Stateless application design
- Database connection pooling
- Concurrent pipeline processing
- Load balancer friendly

### Performance
- Efficient database queries with proper indexing
- Connection pooling and reuse
- Batch processing for large datasets
- Caching strategies for frequently accessed data

### Reliability
- Graceful error handling and recovery
- Circuit breaker patterns for external dependencies
- Retry logic with exponential backoff
- Health checks and monitoring

## Development Workflow

### Local Development
```bash
# Start development environment
docker-compose up -d

# Run locally
make run

# Run tests
make test

# Build for production
make build
```

### Testing Strategy
- Unit tests for individual components
- Integration tests for API endpoints
- End-to-end tests for complete workflows
- Performance tests for scalability validation

### Deployment
- Docker containers for consistent environments
- Kubernetes manifests for orchestration
- CI/CD pipeline integration
- Blue-green deployment support

## Future Enhancements

### Planned Features
- Real-time WebSocket API for live updates
- Advanced ML models for better classification
- Multi-language support for international sources
- GraphQL API for flexible data querying

### Scalability Improvements
- Redis caching layer
- Message queue for async processing
- Microservices decomposition
- Event-driven architecture

### Monitoring Enhancements
- Distributed tracing with Jaeger
- Advanced alerting with PagerDuty
- Custom dashboards with Grafana
- Log aggregation with ELK stack