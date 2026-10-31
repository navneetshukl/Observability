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


# Milestone 2: Run Prometheus and scrape your app

## Learning objectives
- Run Prometheus with Docker Compose and point it at your Go service.
- Understand the pull model, scrape configuration, and target health.
- Use the Prometheus UI to inspect raw time series.

## Requirements

**1. Containerize the app.** Write a `Dockerfile` for `orderly`. A multi-stage build (build with the Go image, run on a small image such as `alpine` or `distroless`) is the production-standard approach.

**2. Write `docker-compose.yml`** with two services:
- `orderly`, built from your Dockerfile, exposing 8080 to your host.
- `prometheus` (`prom/prometheus`), exposing 9090, with your `prometheus.yml` mounted into the container and a named volume for its data directory.

**3. Write `prometheus.yml`** with:
- `global.scrape_interval: 5s` (short for learning; production is usually 15-60s).
- A scrape job `orderly` targeting your app.
- A second scrape job where Prometheus scrapes **itself**.

**4. Keep a traffic loop running** in another terminal so there is data to look at.

## Concepts you should understand

**Pull model.** Prometheus decides when to scrape and what to scrape. Your app only exposes current state at `/metrics` and never knows Prometheus exists. Why production teams like this:
- **Health for free.** If a scrape fails, Prometheus records `up == 0`, so "the target is down" is itself a metric. In push systems, a dead app simply goes silent and you can't tell silence from "no events".
- **Control sits in one place.** Scrape rate and targets are configured centrally, not in every app.
- **Easy debugging.** You can `curl /metrics` yourself and see exactly what Prometheus sees.

Trade-off: short-lived batch jobs may finish before being scraped. That's the one case where the Pushgateway exists. Know that it exists and why it's discouraged for normal services.

**Targets, jobs, instances.** A *target* is one scrapeable endpoint. A *job* is a group of targets with the same role. Prometheus automatically attaches the labels `job` and `instance` to everything it scrapes. This is how, in production, you can ask "error rate across all instances of job `orderly`" or drill into a single pod.

**Service discovery (conceptual).** You'll use `static_configs`. In production, targets come from Kubernetes, Consul, or cloud APIs because instances come and go. Be able to explain why hardcoded targets don't scale.

**The `up` metric and staleness.** Prometheus synthesizes `up{job, instance}` (1 or 0) for every target on every scrape. When a series stops being reported, Prometheus marks it stale so graphs don't show lines for dead targets.

**TSDB basics.** Prometheus stores data in a local time-series database with a retention period (default 15 days). It is designed as a single-node system for recent data. Long-term storage and HA are solved with remote write or tools like Thanos/Mimir. You only need to know that this exists.

**Networking gotcha.** Inside Compose, `localhost` means *the container itself*. Services reach each other by service name.

## Hints
- Prometheus config reference: `scrape_configs`, `job_name`, `static_configs`, `targets`.
- Prometheus image reads its config from `/etc/prometheus/prometheus.yml`. Check the image docs for the data path and flags such as `--config.file` and `--storage.tsdb.retention.time`.
- Compose `volumes:` has two forms: bind mount (for the config) and named volume (for data).
- Use `docker compose logs prometheus` if the container exits. A YAML mistake is the usual cause.
- After changing `prometheus.yml`, you have to reload. Look up the two ways: restarting the container, or the `/-/reload` endpoint (needs `--web.enable-lifecycle`).

## How to verify
1. Open `http://localhost:9090/targets`. Both jobs must show **UP**. Note the "last scrape" time and the scrape duration.
2. In the Graph tab, run `up`. You should see two series. Look at the labels on each.
3. Run `orderly_orders_created_total` and look at the graph. It should be a rising staircase.
4. Run `go_goroutines` and `process_resident_memory_bytes`.
5. `docker compose stop orderly`. After a few seconds `up{job="orderly"}` should drop to 0 and the target shows DOWN with an error. Start it again and watch it recover. Notice what happens to the counter value after the restart.

## Questions to answer
1. Why can't you use `localhost:8080` as the target here? What did you use instead?
2. If the scrape interval is 5s and your app's counter increments 100 times between scrapes, what does Prometheus know about those 100 events? What does it *not* know?
3. What happens to `up` and to your data if Prometheus is down for 10 minutes while your app keeps running?

---

# Milestone 3: RED metrics for HTTP (Histograms, labels, cardinality)

## Learning objectives
- Instrument every HTTP request automatically with middleware.
- Implement the **RED method**: Rate, Errors, Duration.
- Understand Histograms in depth and choose buckets deliberately.
- Understand labels and why high cardinality is the #1 way people break Prometheus.

## Requirements

**1. Write an HTTP middleware** that wraps all your handlers (not `/metrics` itself) and records two metrics:

| Metric | Type | Labels |
|---|---|---|
| `orderly_http_requests_total` | Counter | `method`, `path`, `status` |
| `orderly_http_request_duration_seconds` | Histogram | `method`, `path` |

**2. Label rules**
- `path` must be the **route pattern** (`/orders/{id}`), never the raw URL (`/orders/4821`).
- `status` should be the actual response code as a string (`"201"`, `"500"`).

**3. Choose histogram buckets deliberately** based on your service. Your slowest normal requests are about 300 ms. Pick roughly 8-10 buckets that give good resolution between 10 ms and 1 s, plus some headroom for outliers. Write down your reasoning in a comment.

**4. Make traffic realistic.** Write a small load generator (a Go program, a bash loop, or a tool like `hey`/`k6`) that sends:
- Mostly `POST /orders` with some concurrency.
- Some `GET /orders/{id}` for existing IDs.
- Some `GET /orders/99999999` (404s).
- Continuous traffic for a few minutes.

**5. Cardinality experiment (required).**
- First do it the *wrong* way: temporarily use the raw `r.URL.Path` as the `path` label and send GETs for 500 different IDs.
- Observe the damage, then switch back to the route pattern.

## Concepts you should understand

**RED method.** For any request-driven service, three signals cover most problems: **R**ate (requests/sec), **E**rrors (failed requests/sec or ratio), **D**uration (latency distribution). Almost every backend dashboard at companies using Prometheus starts here. Its companion for infrastructure (CPU, memory, disk, pools) is the USE method (Utilization, Saturation, Errors), which you'll use in Milestones 5 and 9.

**Labels and dimensions.** A label turns one metric into many time series: `orderly_http_requests_total{method="POST",path="/orders",status="500"}` is a different series from the `status="201"` one. Labels let you slice (error rate per endpoint), but each unique combination costs memory and storage in Prometheus.

**Cardinality.** The number of series for a metric = the product of distinct values of each label. If `path` takes 500,000 values because user IDs are embedded, you have 500,000 series per method/status combination. Real-world consequences:
- Prometheus memory grows and queries slow down.
- It can crash with out-of-memory errors.
- In managed services, your bill explodes.

Rule of thumb: **label values must come from a small, bounded set.** Never use user IDs, emails, order IDs, request IDs, raw URLs, or error message text as labels. Those belong in logs or traces.

**Histogram.** A histogram doesn't store individual observations. It stores *counters per bucket*. For one metric you get:
- `_bucket{le="0.1"}`: the cumulative count of observations ≤ 0.1s. The `le` ("less than or equal") buckets are **cumulative**, so the `+Inf` bucket equals the total count.
- `_sum`: the sum of all observed values (lets you compute averages).
- `_count`: the total number of observations.

Why this design: histograms can be **aggregated across instances**. If you have 20 pods, you can sum their bucket counters and compute a fleet-wide p99. You cannot do that with pre-computed per-instance percentiles (the Summary in Milestone 5). The trade-off is that quantiles are estimated by interpolating inside a bucket, so bucket choice controls accuracy. If your SLO is "99% of requests under 300 ms", you want a bucket boundary at exactly 0.3.

**Why not averages?** An average of 100 ms can hide the fact that 1% of users wait 3 seconds. Production teams alert on p95/p99, not on the mean.

**Observing time correctly.** Use seconds as float64 (`time.Since(start).Seconds()`). Prometheus convention is base units, so never milliseconds.

## Hints
- Use `promauto.NewCounterVec` and `promauto.NewHistogramVec`. The `Vec` types are what give you labels, and you call `.WithLabelValues(...)` to pick the series.
- Wrapping the handler: look at how `http.Handler` / `http.HandlerFunc` middleware works. Wrap the whole `mux`, or wrap each route.
- To capture the status code you need a small struct that embeds `http.ResponseWriter` and overrides `WriteHeader`. Remember that if a handler never calls `WriteHeader`, the status is 200.
- To get the route pattern: in Go 1.23+ `r.Pattern` is set after the mux matches, so read it **after** calling `next.ServeHTTP`. It contains the method too (`POST /orders`), so check how to handle that. If you use Gin, `c.FullPath()` does the job.
- Bucket examples to think about: `prometheus.DefBuckets` is generic (5 ms to 10 s). Is it a good fit for you? What would you change?
- For the cardinality experiment, see what happens to the size of `curl localhost:8080/metrics | wc -l`.

## How to verify
1. `curl localhost:8080/metrics | grep orderly_http` and confirm you see `_bucket`, `_sum` and `_count` lines for the histogram.
2. In the Prometheus UI, run these and make sure they return sensible results (you'll learn why they work in Milestone 4, so just try them now):
   - `sum(rate(orderly_http_requests_total[1m]))`
   - `sum(rate(orderly_http_requests_total{status=~"5.."}[1m])) / sum(rate(orderly_http_requests_total[1m]))`
   - `histogram_quantile(0.95, sum by (le) (rate(orderly_http_request_duration_seconds_bucket[1m])))`
3. The error ratio should be close to 5% (a bit lower, because failures only happen on `POST /orders` and not on GETs).
4. p95 for `POST /orders` should be near 280 ms, because of the 10-300 ms uniform delay. If it's way off, check your bucket boundaries.
5. Cardinality experiment: with the raw-URL label, run `count({__name__=~"orderly_http.*"})` and check `prometheus_tsdb_head_series` before and after. Record the numbers.

## Questions to answer
1. Why is `path="/orders/{id}"` fine as a label but `path="/orders/4821"` is not? Name a real label you've seen or would be tempted to add that could cause the same problem.
2. Your histogram has buckets up to 1s, but a request takes 4s. Where is it counted? What does that do to `histogram_quantile(0.99, ...)` if many requests are like that?
3. Why are bucket counters cumulative, and why does that make histograms aggregatable across 20 instances?
4. Why does the status label use `"201"` and `"500"` rather than just "success/failure"? What would you lose?

---

When you finish, send me:
- `docker-compose.yml`, `prometheus.yml`, and the `Dockerfile` (Milestone 2)
- Your middleware and histogram definition, plus the cardinality experiment numbers (Milestone 3)
- Your answers to the questions

I'll review them before we start **Milestone 4: PromQL fundamentals**, where you'll learn why the queries in step 2 above work.