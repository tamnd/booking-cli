---
title: "Introduction"
description: "What booking is and how it is put together."
weight: 10
---

A command line for Booking.com.

booking is a single binary. It reads public Booking.com data over plain HTTPS,
shapes the responses into clean records, and gets out of your way. There is no
API key, so every surface is anonymous. There is nothing to sign up for and
nothing to run alongside it.

## One web plane, two reliability tiers

Booking.com has no free public API, so booking reads the public website. That
website has two tiers, and booking is honest about which is which.

- **The destination estate** is the country, region, city, district, landmark,
  and airport landing pages. These exist to be crawled and read from anywhere.
  They are the reliable backbone and host the `destination`, `destinations`, and
  `properties` commands.
- **The interactive client** is the property page, `reviews`, `search`, and
  `suggest`. These sit behind a bot manager. They work from a residential or
  mobile connection, and are best-effort from a datacenter, where a wall returns
  exit code 4. When that happens, read the destination estate instead, which is
  not gated.

A nightly price is filled only when you pass both `--checkin` and `--checkout`.
It is never invented.

## How it is built

- A **library package** (`booking`) holds the HTTP client and the typed data
  models (Property, Review, Destination, Suggestion, Ref). It paces requests,
  sets an honest User-Agent, and retries the transient failures any public site
  throws under load.
- A **domain** (`booking/domain.go`) declares each operation once on the
  [any-cli/kit](https://github.com/tamnd/any-cli) framework. That single
  declaration becomes a CLI command, an HTTP route, an MCP tool, and a
  resource-URI dereference. It is the one place you add to the tool.
- A thin **`cmd/booking`** hands the assembled app to `kit.Run`, which builds the
  command tree and the serve and mcp surfaces.

## One operation, four surfaces

Because an operation is surface-neutral, the same `property` you run on the
command line is also a route and a tool:

```bash
booking property gb/the-savoy            # the command
booking serve --addr :7777               # GET /v1/property/gb/the-savoy
booking mcp                              # the property tool, over stdio
ant get booking://property/gb/the-savoy  # the URI dereference (via a host)
```

You write the fetch and the record shape; the surfaces come for free.

## Scope

booking is a read-only client over data Booking.com already serves publicly. It
reads that data and shapes it for you. That narrow scope keeps it a single small
binary with no database, no daemon, and no setup.

Next: [install it](/getting-started/installation/), then take the
[quick start](/getting-started/quick-start/).
