---
title: "Configuration"
description: "Environment variables, defaults, and the data directory."
weight: 20
---

booking needs almost no configuration: it runs anonymously against public data
out of the box. There is no API key, because Booking.com has no free public API,
so there are no credential or API-key environment variables to set. The settings
below let you tune the search context, politeness, and storage.

## Search context

These shape what Booking.com returns and what currency and language it uses.

| Setting | Default | Flag |
|---|---|---|
| Locale | `en-us` | `--locale` |
| Currency | `USD` | `--currency` |
| Adults | `2` | `--adults` |
| Children | `0` | `--children` |
| Rooms | `1` | `--rooms` |
| Check-in date | (none) | `--checkin YYYY-MM-DD` |
| Check-out date | (none) | `--checkout YYYY-MM-DD` |

A nightly price is filled only when both `--checkin` and `--checkout` are passed.
It is never invented.

## Requests and cache

| Setting | Default | Flag |
|---|---|---|
| User-Agent | `booking-cli/0.1 (+https://github.com/tamnd/booking-cli)` | `--user-agent` |
| Requests | paced and retried on 429/5xx | `--rate`, `--retries` |
| Per-request timeout | 30s | `--timeout` |
| Cache freshness | `24h0m0s` | `--cache-ttl` |
| On-disk cache | under the data directory | `--no-cache` to bypass |
| Fresh fetch | use the cache | `--refresh` to rewrite it |

`--refresh` fetches fresh and rewrites the cache, ignoring any hit. `--no-cache`
bypasses the cache for the run without rewriting it.

## The data directory

Caches and any record store live under one data directory, chosen in this order:

1. `--data-dir`
2. `BOOKING_DATA_DIR`
3. `$XDG_DATA_HOME/booking`
4. `~/.local/share/booking`

## Environment variables

Every flag has an environment fallback, prefixed `BOOKING_` in upper case with
dashes as underscores. For example:

```bash
export BOOKING_CURRENCY=GBP    # same as --currency GBP
export BOOKING_RATE=1s         # same as --rate 1s
export BOOKING_DATA_DIR=~/data/booking
```

Flags win over environment variables, which win over the built-in defaults. There
are no API-key variables to set, because every surface is anonymous.

## Sending records to a store

`--db` tees every emitted record into a store as a side effect of reading, so a
session fills a local database without a separate import step:

```bash
booking properties city/us/orlando --db out.db        # SQLite file
booking properties city/us/orlando --db 'postgres://...'
```
