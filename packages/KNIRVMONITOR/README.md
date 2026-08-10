# network_monitor

HTTP metrics aggregation service for the KNIRV Network. Extends the existing alertmanager-config CLI into a full Prometheus-exposed metrics server with local process observability and health endpoints.

## What It Does

- Exposes `/healthz`, `/readyz`, `/metrics`, and `/api/v1/status`
- Collects local process metrics (CPU, memory, disk, goroutines, uptime) via gopsutil
- Registers Prometheus gauges/counters on a dedicated registry (avoids double-registration conflicts)
- Serves as the backend for the **Network Monitor** admin tab in the KNIRVSERVER dashboard

## Endpoints

| Path | Method | Description |
|------|--------|-------------|
| `/healthz` | GET | Liveness probe |
| `/readyz` | GET | Readiness probe |
| `/metrics` | GET | Prometheus exposition format |
| `/api/v1/status` | GET | JSON status with process metrics |

## Running

```bash
go run ./cmd/server/main.go --port 9091
```

Configuration via flags:

- `--port` — HTTP listen port (default `9091`)
- `--prometheus-url` — Prometheus upstream URL
- `--grafana-url` — Grafana upstream URL
- `--scrape-interval` — Internal metrics collection interval
- `--request-timeout` — HTTP request timeout for external probes

## Prometheus Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `network_monitor_process_cpu_seconds_total` | Gauge | CPU usage percent |
| `network_monitor_process_memory_bytes` | Gauge | Memory used in bytes |
| `network_monitor_process_disk_total_bytes` | Gauge | Total disk capacity |
| `network_monitor_process_disk_used_bytes` | Gauge | Disk used in bytes |
| `network_monitor_process_goroutines` | Gauge | Active goroutine count |
| `network_monitor_process_uptime_seconds` | Gauge | Process uptime |
| `network_monitor_scrape_errors_total` | Counter | Collection errors |

## Frontend Integration

The Network Monitor tab lives in `KNIRV_NETWORK/packages/KNIRVSERVER/frontend/src/components/dashboard/dashboard-wrapper.tsx`. It is admin-gated via `RoleGuard` and contains seven sub-views:

- **Health** — Host CPU/memory/disk + service up/down
- **Metrics** — KNIRVBASE/KNIRVCHAIN Prometheus metrics
- **Routes** — KNIRVGATEWAY proxy route table
- **Grafana** — Embedded dashboard via iframe
- **Prometheus** — Targets page via iframe
- **Onboarding** — Operator applications and user accounts
- **Alerts** — Unified alert feed

## Testing

```bash
go test -v ./...
go vet ./...
```
