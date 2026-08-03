# Overview
Go service for currency quotes: request an update, fetch the result by ID. Background worker, PostgreSQL, HTTP/JSON API.

Supported currencies: CAD, EUR, GBP, MXN, USD.

The service provides quotes with a precision of 6 decimal places.

## Running the project

There are two ways to run this project:

1. **Developing** — prerequisites: Docker, Go 1.25+, goose.
2. **Just running it** — prerequisites: Docker, goose.

Migrations need the [goose](https://github.com/pressly/goose) CLI either way — either `go install github.com/pressly/goose/v3/cmd/goose@latest` (needs Go) or a prebuilt binary from goose's [releases page](https://github.com/pressly/goose/releases) (doesn't).

Both scenarios share the same first steps:

```
git clone https://github.com/spenderella/currency-quotes-service.git
cd currency-quotes-service
cp env.example .env             # fill in POSTGRES_*, HTTP_SERVER_ADDRESS=:8080, etc.
docker-compose up -d postgres   # start PostgreSQL
make migrate-up                 # apply migrations
```

`.env` is read by the app as an actual file at startup (via `godotenv`), not just from the process environment — it's gitignored and never baked into any image, only ever bind-mounted or read straight off disk.

### 1. Developing

```
make run   # go run cmd/server/main.go — fastest feedback loop, no image build/rebuild per change
```

#### Publishing a new image

Once a change is ready to ship, bump the version in `docker-compose.yaml`'s `app.image` (e.g. `1.0.0` → `1.0.1`). GHCR doesn't enforce immutable tags, so re-pushing an already-published version would silently overwrite it. Then build and push:

```
docker compose build app
docker compose push app
```

### 2. Just running the service

No Go toolchain needed — the app runs from the pre-built image in the registry instead of building locally.

```
docker compose pull app   # fetch the pre-built image (bypasses the local Dockerfile build)
docker-compose up -d      # bring up the app too, now that the schema exists
```

## API

All endpoints are JSON. Non-2xx responses have the shape `{"error": "..."}`.

**`POST /quotes`** — request a quote update. Runs in the background; the handler doesn't wait on the provider. Requires a client-generated `Idempotency-Key` header — replaying the same key returns the same update instead of creating a duplicate.

```
curl -X POST localhost:8080/quotes \
  -H 'Idempotency-Key: 3b1f6e9a-...' \
  -H 'Content-Type: application/json' \
  -d '{"base_currency": "EUR", "quote_currency": "MXN"}'
# 202 {"id": "..."}
```

**`GET /quotes/{id}`** — fetch an update by ID, in whatever status it's currently in (`pending`/`in_progress`/`done`/`failed`). `rate`/`provider_time`/`fetched_at` are only present once the update is `done`.

```
curl localhost:8080/quotes/3b1f6e9a-...
# 200 {"id": "...", "base_currency": "EUR", "quote_currency": "MXN", "status": "done", "rate": "21.5", "provider_time": "...", "fetched_at": "..."}
```

**`GET /quotes/latest?base=EUR&quote=MXN`** — the most recent successfully completed (`done`) update for a currency pair. `404` if none exists yet.

```
curl 'localhost:8080/quotes/latest?base=EUR&quote=MXN'
```

`404` is also returned by `GET /quotes/{id}` for an unknown ID; `422` is returned by both `POST /quotes` and `GET /quotes/latest` for a currency outside the supported whitelist.

## Rate provider

Rates are sourced from Banco de México (Banxico) reference rates via [Frankfurter](https://frankfurter.dev) (`providers=BANXICO`). Central bank reference rates are published once per business day, so rates only actually change on that cadence regardless of how often an update is requested. `RatesProvider` is defined as an interface, so a different provider can be swapped in if a task needs more frequent updates or coverage from another source.

### Retry policy

`GetRate` retries transient failures with exponential backoff: network errors, failures reading the response body, and 5xx responses are retried; 4xx responses and response decode/parse errors are not, since retrying can't fix a malformed request or response. Up to `PROVIDER_MAX_RETRIES` attempts (default 3), with the delay doubling from `PROVIDER_RETRY_BASE_DELAY_MS` (default 200ms) after each attempt. A cancelled or expired context stops retrying immediately.

## Background worker

Rates update asynchronously, independently of request handling.

A ticker wakes on a fixed interval and claims pending update tasks with `SELECT ... FOR UPDATE SKIP LOCKED`, atomically flipping their status to `in_progress` in the same transaction, then hands them to a fixed-size pool of worker goroutines over a channel.

This mirrors how production job queues are built directly on top of Postgres, with no separate broker: `SKIP LOCKED` lets concurrent claimers grab disjoint batches of rows instead of blocking each other, the bounded worker pool caps how many requests hit the rate provider at once, and a short tick interval keeps latency low — all without ever blocking the HTTP handler. A one-time recovery pass at startup resumes any task left `in_progress` by a crash.

This production-style design trades a bit of complexity for low latency and bounded provider load. At lower traffic, a plain ticker without a worker pool, or a goroutine per request, would be simpler and just as sufficient.

## Tests

Unit tests cover `CurrencyService` and `QuoteService` (`internal/service/*_test.go`): currency whitelist loading, pair validation, mapping a repository "not found" to the service-level error, and both the success and failure paths of processing a claimed rate-update task (provider error, mark-done/mark-failed persistence errors). Repository, currency-service, and provider dependencies are mocked with `go.uber.org/mock` (mocks generated into `internal/service/mocks` via `//go:generate`); assertions use `testify`.

```
go test ./...
```

Integration tests against a real PostgreSQL instance are not written yet.






