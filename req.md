# Project: "Orderly" — a small orders API with full observability

You'll build a small Go HTTP service (an orders API with PostgreSQL behind it), plus a load generator, and then build the monitoring stack around it. The domain is deliberately boring so all your attention goes to the observability.

**Final architecture (Docker Compose):**
- `orderly` (your Go app, exposes `/metrics`)
- `postgres` + `postgres_exporter`
- `cadvisor` (container CPU/memory)
- `prometheus` (scrapes everything, evaluates rules)
- `alertmanager`
- `grafana`
- A load generator (Go program or `hey`/`k6`) that creates realistic traffic, including errors and slow requests

## Milestone roadmap

| # | Milestone | Main concepts |
|---|-----------|---------------|
| 1 | Instrument a Go service | Metrics, Counter, `/metrics` format, default Go runtime metrics |
| 2 | Prometheus + scraping | Pull model, scrape config, targets, `up`, Compose |
| 3 | RED metrics for HTTP | Histogram, labels, middleware, cardinality |
| 4 | PromQL fundamentals | `rate`, `histogram_quantile`, aggregation, `by`/`without` |
| 5 | Gauges, Summaries, dependency metrics | In-flight requests, DB pool stats, Summary vs Histogram |
| 6 | Grafana dashboards | Data source, panels, variables, RED dashboard |
| 7 | Recording and alerting rules | Pre-computation, `for`, severity, SLO-style alerts |
| 8 | Alertmanager and Grafana alerts | Routing, grouping, silencing, contact points |
| 9 | Exporters and resource monitoring | postgres_exporter, cAdvisor, USE method, cardinality blow-up experiment |

We'll do one at a time. When you finish a milestone, paste your code and config here and I'll review it before we move on.

---

# Milestone 1: Instrument a Go service

## Learning objectives
- Understand what a metric is and what the Prometheus text exposition format looks like.
- Expose `/metrics` from a Go app using the official client library.
- Create and use a Counter, and see what you get for free from Go runtime metrics.

## Requirements

**Project setup**
1. Create a Go module `orderly` with this layout (suggested): `cmd/orderly/main.go`, `internal/handlers`, `internal/metrics`.
2. Use the standard library `net/http` or Gin, your choice. Stay in the same framework for the whole project.

**Endpoints (in-memory storage is fine for now; Postgres comes later)**
- `POST /orders`: creates an order (JSON body with `item` and `amount`), returns 201.
- `GET /orders/{id}`: returns the order, or 404.
- `GET /health`: returns 200.
- `GET /metrics`: Prometheus endpoint.

**Behavior to build in on purpose** (you'll need these for later milestones)
- Add a random artificial delay of 10-300 ms in `POST /orders`.
- Make about 5% of `POST /orders` calls fail with a 500.

**Metrics**
1. Expose the default Go runtime and process metrics via `/metrics`.
2. Create a custom Counter `orderly_orders_created_total` incremented on every successful order creation.
3. Create a custom Counter `orderly_orders_failed_total` incremented on every failed one.

Follow Prometheus naming conventions: `namespace_subsystem_name_unit`, counters end in `_total`, base units (seconds, bytes) only.

## Concepts you should understand

**What a metric is.** A metric is a numeric measurement identified by a name plus a set of labels, sampled over time. Prometheus doesn't store events; it stores a time series per unique name+label combination: `(timestamp, value)` pairs. This is why metrics are cheap compared to logs: you store aggregates, not every event. Production use: metrics tell you *that* something is wrong and roughly *where*; logs and traces tell you *why*.

**Counter.** A value that only goes up (or resets to zero on process restart). You almost never look at the raw value. You look at its rate of change, which you'll do in PromQL later. Why it must be monotonic: Prometheus can detect resets and still compute correct rates, and a scrape that's missed doesn't lose information, because the next scrape's value includes everything in between. That is the big design win over pushing individual events.

**Exposition format.** `/metrics` is plain text: `# HELP`, `# TYPE`, then lines like `name{label="x"} value`. Read it by eye; it's your debugging tool for the rest of the project.

**Default registry and runtime metrics.** The client library's default registry already includes the Go collector (`go_goroutines`, `go_memstats_*`, `go_gc_duration_seconds`, ...) and process collector (`process_cpu_seconds_total`, `process_resident_memory_bytes`, ...). These are what you'll later use for goroutine-leak and memory alerts, and you get them with no extra code.

## Hints
- Package: `github.com/prometheus/client_golang/prometheus`, plus `promauto` for concise registration and `promhttp` for the handler.
- Look at what `promhttp.Handler()` returns and how it relates to the default registry.
- Define metrics once, at package level or in a struct created at startup. Never create a metric inside a request handler. Think about why that would be a bug.
- Keep metric definitions in `internal/metrics` so handlers import them rather than own them.

## How to verify
1. `go run ./cmd/orderly`, then `curl localhost:8080/metrics`. Confirm you see `go_goroutines` and `process_resident_memory_bytes`.
2. Create 20 orders in a loop, e.g. `for i in $(seq 20); do curl -s -XPOST localhost:8080/orders -d '{"item":"book","amount":10}'; done`.
3. Run `curl -s localhost:8080/metrics | grep orderly_` and confirm the two counters add up to 20 and the failure counter is non-zero (you may need more requests, since failures are about 5%).
4. Restart the app and confirm the counters reset to 0. Be ready to explain in your own words why that's okay.

## Questions to answer when you submit
1. What is the difference between `orderly_orders_created_total` and a gauge like "current orders in the system"? Which one would you use for each?
2. Why does Prometheus prefer you to expose cumulative totals instead of "orders in the last minute"?
