# Overview
Go service for currency quotes: request an update, fetch the result by ID. Background worker, PostgreSQL, HTTP/JSON API.

Supported currencies: CAD, EUR, GBP, MXN, USD.

The service provides quotes with a precision of 6 decimal places.

## Rate provider

Rates are sourced from Banco de México (Banxico) reference rates via [Frankfurter](https://frankfurter.dev) (`providers=BANXICO`). Central bank reference rates are published once per business day, so rates only actually change on that cadence regardless of how often an update is requested. `RatesProvider` is defined as an interface, so a different provider can be swapped in if a task needs more frequent updates or coverage from another source.

## Background worker

Rates update asynchronously, independently of request handling.

A ticker wakes on a fixed interval and claims pending update tasks with `SELECT ... FOR UPDATE SKIP LOCKED`, atomically flipping their status to `in_progress` in the same transaction, then hands them to a fixed-size pool of worker goroutines over a channel.

This mirrors how production job queues are built directly on top of Postgres, with no separate broker: `SKIP LOCKED` lets concurrent claimers grab disjoint batches of rows instead of blocking each other, the bounded worker pool caps how many requests hit the rate provider at once, and a short tick interval keeps latency low — all without ever blocking the HTTP handler. A one-time recovery pass at startup resumes any task left `in_progress` by a crash.

This production-style design trades a bit of complexity for low latency and bounded provider load. At lower traffic, a plain ticker without a worker pool, or a goroutine per request, would be simpler and just as sufficient.






