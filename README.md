# portfolio-monitor

Ultra-lightweight centralized telemetry & Prometheus metrics aggregator for `churma.codes`.

## Features
- **Zero-Dependency Core**: Standard Go library runtime (~10MB binary, ~8MB RAM).
- **Ingestion API**: `POST /api/events` (and `/api/metrics`) accepts single or batch telemetry events from portfolio backends.
- **Prometheus Exporter**: `GET /metrics` exposes standard Prometheus text exposition format (counters, histograms, summaries).
- **Health & Inspection**: `GET /health` and `GET /api/stats` for live JSON status and recent events inspection.

## API Contracts

### Ingestion: `POST /api/events`
```json
{
  "source": "laravel",
  "type": "http_request",
  "endpoint": "/api/projects",
  "status_code": 200,
  "duration_ms": 14.5,
  "metadata": {
    "ip": "127.0.0.1"
  }
}
```

Response (`202 Accepted`):
```json
{
  "status": "recorded",
  "count": 1
}
```
