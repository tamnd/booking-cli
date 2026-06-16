---
title: "booking"
description: "A command line for Booking.com."
heroTitle: "booking, from the command line"
heroLead: "A command line for Booking.com. One pure-Go binary, no API key, output that pipes into the rest of your tools, and a resource-URI driver other programs can address."
heroPrimaryURL: "/getting-started/quick-start/"
heroPrimaryText: "Get started"
---

`booking` reads public Booking.com data over plain HTTPS, shapes it into clean
records, and gets out of your way. There is no API key. Every surface is
anonymous.

```bash
booking suggest orlando                  # autocomplete places and hotels
booking destination country/us           # one node of the geographic tree
booking properties city/us/orlando       # the hotels on a city landing page
booking property gb/the-savoy            # one property in full
booking serve --addr :7777               # the same operations over HTTP
```

Output adapts to where it goes: an aligned table on your terminal, JSONL the
moment you pipe it somewhere.

## One web plane, two reliability tiers

Booking.com has no free public API, so `booking` reads the public website. That
website has two tiers, and `booking` tells you which is which.

- **The destination estate** is the country, region, city, district, landmark,
  and airport landing pages. These exist to be crawled and read from anywhere.
  They are the reliable backbone and the home of `destination`, `destinations`,
  and `properties`.
- **The interactive client** is the property page, `reviews`, `search`, and
  `suggest`. These sit behind a bot manager. They work from a residential or
  mobile connection, and are best-effort from a datacenter, where a wall returns
  exit code 4.

A nightly price is filled only when you pass both `--checkin` and `--checkout`.
It is never invented.

## Two ways to use it

- **As a command** for reading Booking.com by hand or in a script. Start with
  the [quick start](/getting-started/quick-start/).
- **As a resource-URI driver** so a host like
  [ant](https://github.com/tamnd/ant) can address Booking.com as `booking://`
  URIs and follow links across sites. See
  [resource URIs](/guides/resource-uris/).

Both are the same code: one operation, declared once, is a CLI command, an HTTP
route, an MCP tool, and a URI dereference.

## Where to go next

- New here? Read the [introduction](/getting-started/introduction/), then the
  [quick start](/getting-started/quick-start/).
- Installing? See [installation](/getting-started/installation/).
- Doing a specific job? The [guides](/guides/) are task-first.
- Need every flag? The [CLI reference](/reference/cli/) is the full surface.
