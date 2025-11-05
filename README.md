# Go Gin Observability Template

A production-ready Go microservice template with Gin framework and full observability stack (Grafana, Prometheus, Loki, Tempo, Pyroscope).

## Features

- **Gin Web Framework** with clean architecture
- **PostgreSQL** with migrations and seeding
- **JWT Authentication**
- **OpenTelemetry** tracing and metrics
- **Structured Logging** with Loki
- **Continuous Profiling** with Pyroscope
- **Docker Compose** for development and staging
- **Hot Reload** with Air

## Prerequisites

- Go 1.25+
- Docker & Docker Compose
- Make

## Quick Start

### 1. Environment Setup

Create `.env` file:

```env
APP_NAME=go-gin-clean-starter
APP_ENV=development
APP_PORT=8080

DB_USER=postgres
DB_PASS=postgres
DB_NAME=myapp
DB_PORT=5432

JWT_SECRET=change-this-secret

OTEL_SAMPLING_RATE=1.0
ENABLE_PROFILING=true
```

### 2. Run Development Environment

```bash
# Start all services
make dev-up
```

Access services:

- API: http://localhost:8080
- Grafana: http://localhost:3000 (admin/admin)
- Prometheus: http://localhost:9090
- Pyroscope: http://localhost:4040

## Development

```bash
# Run locally
make run

# Build binary
make build

# Run migrations
make migrate

# Run tests
make test

# Test coverage
make test-coverage

# Benchmarks
make bench
```

## Project Structure

```
├── cmd/                    # Application entrypoint
├── config/                 # Configuration
├── database/               # Migrations and seeds
├── docker/                 # Docker configs for observability stack
├── middlewares/            # HTTP middlewares
├── modules/                # Feature modules (e.g., account)
├── pkg/                    # Shared packages
│   ├── apm/               # Metrics
│   ├── logger/            # Logging
│   ├── telemetry/         # OpenTelemetry
│   └── tracing/           # Tracing utilities
├── providers/              # Dependency injection
└── script/                 # Utility scripts
```

## Module Management

```bash
# Create new module
make module name=user

# Rename Go module path
make rename name=github.com/yourorg/yourproject
```

## Observability

### Metrics

Prometheus metrics available at `/metrics` endpoint.

### Logging

Structured JSON logs with request correlation:

```json
{
  "time": "2024-01-01T12:00:00Z",
  "level": "INFO",
  "request_id": "abc-123",
  "method": "GET",
  "path": "/api/users",
  "status": 200
}
```

### Tracing

OpenTelemetry traces for all HTTP requests and database queries.

### Profiling

Pyroscope captures CPU, memory, and goroutine profiles continuously.

## Staging Deployment

```bash
# Start staging with Nginx
make staging-up

# Stop staging
make staging-down
```

## Available Commands

Run `make help` to see all available commands.

## License

MIT
