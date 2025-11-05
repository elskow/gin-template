# Observability Stack

This project includes a complete Grafana observability stack for monitoring, logging, and tracing.

## Components

### Grafana
- **URL**: http://localhost:3000
- **Username**: admin
- **Password**: admin
- Visualization and dashboards for metrics, logs, and traces

### Prometheus
- **URL**: http://localhost:9090
- Metrics collection and storage
- Scrapes metrics from the application

### Loki
- **URL**: http://localhost:3100
- Log aggregation system
- Collects logs from Docker containers via Alloy

### Tempo
- **URL**: http://localhost:3200
- **OTLP gRPC**: localhost:4317
- **OTLP HTTP**: localhost:4318
- Distributed tracing backend
- Stores and queries trace data

### Alloy
- **URL**: http://localhost:12345
- Grafana's telemetry collector
- Collects logs, metrics, and traces
- Routes data to appropriate backends

## Quick Start

### Development Mode (PostgreSQL + Observability)
```bash
make dev-up
make dev-grafana  # Opens Grafana in browser
```

### Staging Mode (Full Stack + Observability)
```bash
make staging-init
make staging-grafana  # Opens Grafana in browser
```

## Accessing Services

Once the stack is running, you can access:

1. **Grafana Dashboard**: http://localhost:3000
   - Pre-configured with datasources for Prometheus, Loki, and Tempo
   - Correlation between logs, metrics, and traces

2. **Prometheus**: http://localhost:9090
   - Query metrics directly
   - Check targets status

3. **Direct API Access**:
   - Loki: http://localhost:3100
   - Tempo: http://localhost:3200

## Features

### Logs
- Automatic collection from Docker containers
- Structured JSON logs
- Request ID tracking for correlation
- Link from logs to traces

### Metrics
- Application metrics scraping
- System metrics from containers
- Custom metrics support

### Traces
- OpenTelemetry Protocol (OTLP) support
- Distributed tracing across services
- Service graph generation
- Link from traces to logs and metrics

### Correlation
All three signals (logs, metrics, traces) are correlated:
- Logs contain `request_id` that links to traces
- Traces contain span attributes that link to logs
- Metrics contain labels that link to services

## Configuration Files

- `docker/loki/loki-config.yml` - Loki configuration
- `docker/tempo/tempo-config.yml` - Tempo configuration
- `docker/prometheus/prometheus.yml` - Prometheus scrape config
- `docker/alloy/config.alloy` - Alloy telemetry collection
- `docker/grafana/provisioning/` - Grafana datasources and dashboards

## Volumes

Data is persisted in Docker volumes:
- `loki-data` - Log data
- `tempo-data` - Trace data
- `prometheus-data` - Metrics data
- `grafana-data` - Grafana dashboards and settings

## Stopping Services

```bash
# Stop development
make dev-down

# Stop staging
make staging-down
```

## Next Steps

After setting up the observability stack:

### 1. Application is Pre-instrumented

The application is already instrumented with:

- **OpenTelemetry Tracing**: Automatic span creation for all HTTP requests via `otelgin.Middleware`
- **Structured Logging**: JSON logs with request_id correlation
- **Metrics Export**: Prometheus metrics at `/metrics` endpoint

### 2. Verify Instrumentation

Start the application and make a test request:

```bash
# Start dev environment
make dev-up
make run

# In another terminal, make a test request
curl http://localhost:8888/health

# Check logs - you'll see JSON with request_id
# {"time":"...","level":"INFO","msg":"incoming request","request_id":"...","method":"GET","path":"/health"}
```

### 3. View in Grafana

1. Open Grafana: http://localhost:3000 (admin/admin)
2. Go to "Explore"
3. Select **Loki** datasource - View application logs
4. Select **Tempo** datasource - View distributed traces
5. Select **Prometheus** datasource - View metrics

### 4. Correlation Example

When you make a request:
- **Logs** will show the request_id
- **Traces** will show the span with the same trace_id
- **Metrics** will show request counts and durations

You can click on a trace ID in Loki logs to jump directly to the trace in Tempo!

### 5. Create Custom Dashboards

Create dashboards in Grafana to visualize:
- Request rates and latencies
- Error rates
- Database query performance
- Custom business metrics
