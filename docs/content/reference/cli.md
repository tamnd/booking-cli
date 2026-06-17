---
title: "CLI"
description: "Every command and subcommand, with the flags that matter."
weight: 10
---

```
booking <command> [arguments] [flags]
```

Run `booking <command> --help` for the full flag list on any command. This page
is the map.

## Commands

The read commands split across the two reliability tiers. The destination estate
is reliable and reads from anywhere. The interactive client is best-effort and
can hit a bot wall (exit code 4) from a datacenter.

| Command | Tier | What it does |
|---|---|---|
| `search <destination>` | best-effort | Search properties by free-text destination. Emits Property cards |
| `property <ref>` | best-effort | Show one property by id or `/hotel/` URL. Emits one Property |
| `reviews <ref>` | best-effort | List a property's reviews. Emits Review records |
| `destination <ref>` | reliable | Show one destination node. Emits one Destination |
| `destinations <ref>` | reliable | List a destination's child nodes. Emits Destination records |
| `properties <ref>` | reliable | List the properties on a destination landing page. Emits Property records |
| `suggest <prefix>` | best-effort | Autocomplete destinations and properties for a prefix. Emits Suggestion records |
| `sitemaps` | reliable | List the sitemap indexes Booking advertises in robots.txt. Emits SitemapIndex records, the root of the crawl root |
| `sitemap <kind>` | reliable | Enumerate a kind's landing pages from Booking's sitemaps. Emits Seed records, the crawl root |
| `ref id <ref>` | offline | Classify any Booking.com URL, path, or id into its (kind, id). Emits a Ref |
| `ref url <kind> <id>` | offline | Build the canonical URL for a (kind, id). Emits a Ref |
| `serve [--addr]` | | Serve the operations over HTTP as NDJSON |
| `mcp` | | Run as an MCP server over stdio |
| `version` | | Print the version and exit |

For `ref url`, `kind` is `property`, `destination`, `search`, `sitemap`, or
`sitemaps`.

`sitemaps` reads robots.txt, the master list, and emits one SitemapIndex per
advertised index: the index URL, the kind it enumerates, and the category of page
it covers (place, property, reviews, attraction, beach, theme, car, flight,
article, other). Booking advertises a few hundred indexes, far more than the
handful of accommodation kinds, so `sitemaps` is how you discover every backbone
the site publishes rather than guessing names. Each record's `seeds_ref` is the
kind to hand to `sitemap`.

For `sitemap`, `kind` is any advertised kind, for example `country`, `hotel`,
`hotel-review`, or `themed-city-ski`. Run `sitemaps` to list them all. Booking
publishes a per-kind sitemap index that lists per-language shards, and each shard
enumerates every landing page of that kind. `sitemap` walks the index, reads the
shards for `--locale`, and emits a Seed per page. A Seed carries the edge into the
rest of the graph when the page maps to a record: a place seed fills
`destination`, a hotel or hotel-review seed fills `property`. A page outside the
accommodations graph (an attraction, beach, or themed list) still comes back as a
Seed with its URL and lastmod and no edge, so the inventory stays complete.
Because a Seed needs no prior id, a crawl can start from `sitemaps`, walk each
index into its seeds, and then follow the record links to reach the rest of the
public site.

## Reference forms

The commands accept these reference forms:

- **Property id**: `<cc>/<slug>`, for example `gb/the-savoy`, or a
  `/hotel/<cc>/<slug>.html` URL (an optional `.<locale>` may sit before `.html`).
- **Destination ref**: `<kind>/<cc>[/<slug>]`, for example `country/us`,
  `region/us/florida`, `city/us/orlando`, `district/gb/soho-london`, or the
  landing-page URL.
- **Search**: the free-text term, or a `/searchresults.html?ss=<term>` URL.

## Global flags

These are shared by every operation, so they work the same on every command.

| Flag | Meaning |
|---|---|
| `-o, --output` | Output format: `auto`, `table`, `markdown`, `list`, `json`, `jsonl`, `csv`, `tsv`, `url`, `raw` |
| `--fields` | Comma-separated columns to keep |
| `--template` | Go text/template applied per record |
| `--no-header` | Omit the header row in `table` and `csv` |
| `-n, --limit` | Stop after N records (0 means no limit) |
| `--rate` | Minimum delay between requests |
| `--retries` | Retry attempts on rate limit or 5xx |
| `--timeout` | Per-request timeout |
| `--data-dir` | Override the data directory |
| `--no-cache` | Bypass on-disk caches |
| `--db` | Tee every record into a store (e.g. `out.db`, `postgres://...`) |
| `-v, --verbose` | Increase verbosity (repeatable) |
| `-q, --quiet` | Suppress progress output |
| `--color` | `auto`, `always`, or `never` |

## Booking-specific flags

| Flag | Default | Meaning |
|---|---|---|
| `--user-agent` | `booking-cli/0.1 (+https://github.com/tamnd/booking-cli)` | The User-Agent sent on every request |
| `--locale` | `en-us` | The Booking.com locale |
| `--currency` | `USD` | The currency for prices |
| `--checkin` | (none) | Check-in date, `YYYY-MM-DD` |
| `--checkout` | (none) | Check-out date, `YYYY-MM-DD` |
| `--adults` | `2` | Number of adults |
| `--children` | `0` | Number of children |
| `--rooms` | `1` | Number of rooms |
| `--cache-ttl` | `24h0m0s` | How long a cached response stays fresh |
| `--refresh` | | Fetch fresh and rewrite the cache, ignoring any hit |

A nightly price is filled only when both `--checkin` and `--checkout` are passed.

See [output formats](/reference/output/) for what `-o`, `--fields`, and
`--template` produce, and [configuration](/reference/configuration/) for
environment variables and defaults.
