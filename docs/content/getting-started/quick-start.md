---
title: "Quick start"
description: "Fetch your first record with booking."
weight: 30
---

Once `booking` is on your `PATH`, start on the reliable tier: the destination
estate. These landing pages read from anywhere, so they are the best place to
begin.

```bash
booking destination country/us           # one node of the geographic tree
```

By default you get an aligned table. Ask for JSON when you want to pipe it:

```bash
$ booking destination country/us -o json
[
  {
    "id": "country/us",
    "name": "United States",
    "kind": "country",
    "country": "us",
    "property_count": 1234567,
    "url": "https://www.booking.com/country/us.html"
  }
]
```

## Walk the destination estate

A destination links down to its child nodes and across to the properties on its
landing page. Each of these is reliable and reads from anywhere:

```bash
booking destinations country/us          # the country's child regions and cities
booking properties city/us/orlando       # the hotels on a city landing page
booking properties city/us/orlando -n 10 # the first ten
```

## Start a crawl from the sitemap

`sitemap` reads Booking's own published sitemaps and is the crawl root: it lists
every landing page of a kind with no prior id, so you do not need to know a single
country or hotel to begin. Each row is a Seed that points into the rest of the
graph.

```bash
booking sitemap country                  # every country landing page
booking sitemap hotel -n 50              # the first fifty hotel pages
booking sitemap city -o jsonl | jq .destination
```

A place seed fills `destination` and a hotel seed fills `property`, so you can
walk straight from a seed into the full record and then follow its links.

## Shape the output

The same flags work on every command:

```bash
booking properties city/us/orlando --fields id,name,stars,rating
booking property gb/the-savoy --template '{{.Name}} {{.City}}'
booking destinations country/us -o jsonl | jq .id
```

`-o` takes `table`, `markdown`, `list`, `json`, `jsonl`, `csv`, `tsv`, `url`, or
`raw`. Left to `auto`, it prints a table to a terminal and JSONL into a pipe, so
the same command reads well by hand and parses cleanly downstream. See
[output formats](/reference/output/) for the full contract.

## The best-effort tier

`property`, `reviews`, `search`, and `suggest` read the interactive client, which
sits behind a bot manager. They work from a residential or mobile connection:

```bash
booking suggest orlando                  # autocomplete places and hotels
booking property gb/the-savoy            # one property in full
booking reviews gb/the-savoy             # that property's reviews
booking search Orlando                   # search by free-text destination
```

From a datacenter these can hit a wall that returns exit code 4. When that
happens, read the destination estate instead, or retry from a residential or
mobile connection. See [troubleshooting](/reference/troubleshooting/).

## Fill a nightly price

A price is filled only when you pass both `--checkin` and `--checkout`:

```bash
booking search Orlando --checkin 2026-07-01 --checkout 2026-07-04
booking property gb/the-savoy --checkin 2026-07-01 --checkout 2026-07-04 \
  --currency GBP --adults 2 --rooms 1
```

## Refer to anything

`ref` classifies and builds Booking.com references offline, with no network:

```bash
booking ref id https://www.booking.com/hotel/gb/the-savoy.html  # to (kind, id)
booking ref url property gb/the-savoy                           # to a URL
```

## Serve it instead

The same operations are available over HTTP and to agents over MCP:

```bash
booking serve --addr :7777 &
curl -s 'localhost:7777/v1/property/gb/the-savoy'   # NDJSON, one record per line
booking mcp                                         # MCP over stdio
```

## What to do next

Follow the record graph: a sitemap seed fans into a destination or a property; a
suggestion fans into a search, a place, or a hotel; a search card walks through to
the full property; a property reaches its reviews and the city it sits in; a
destination climbs to its parent, descends to its children, and lists its
properties. Starting from `sitemap` and following these links reaches the whole
public estate. The [guides](/guides/) cover the common jobs.
