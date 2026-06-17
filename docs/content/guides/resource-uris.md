---
title: "Resource URIs"
description: "Use booking as a database/sql-style driver so a host program can address Booking.com as booking:// URIs."
weight: 20
---

`booking` is a command line, but the `booking` Go package is also a small driver
that makes Booking.com addressable as a resource URI. A host program registers it
the way a program registers a database driver with `database/sql`, then
dereferences `booking://` URIs without knowing anything about how Booking.com is
fetched.

The host that does this today is [ant](https://github.com/tamnd/ant), a single
binary that puts one URI namespace over a family of site tools. The examples
below use `ant`; any program that links the package gets the same behaviour.

## Mounting the driver

A host enables the driver with one blank import, exactly like
`import _ "github.com/lib/pq"`:

```go
import _ "github.com/tamnd/booking-cli/booking"
```

The package's `init` registers a domain with the scheme `booking` for the host
`www.booking.com`. The standalone `booking` binary does not change.

## The URI scheme

A URI is `booking://<type>/<id>`. The id is the same reference each command takes.

| URI                                      | What it is                                  |
| ---------------------------------------- | ------------------------------------------- |
| `booking://property/gb/the-savoy`        | one property, keyed by `<cc>/<slug>`        |
| `booking://reviews/gb/the-savoy`         | that property's reviews                     |
| `booking://destination/country/us`       | one destination node                        |
| `booking://destinations/country/us`      | a destination's child nodes                 |
| `booking://properties/city/us/orlando`   | the hotels on a destination landing page    |
| `booking://search/Orlando`               | a free-text destination search              |
| `booking://sitemaps`                      | every sitemap index Booking advertises      |
| `booking://sitemap/country`               | the country landing pages, the crawl root   |

```bash
ant get booking://property/gb/the-savoy      # the property record
ant cat booking://property/gb/the-savoy      # just the description
ant url booking://destination/country/us     # the live https URL
ant ls  booking://properties/city/us/orlando # the hotels on a city page
```

## The record graph

Records connect into one graph, and every edge is a field that points at another
addressable record:

- A **suggestion** fans out: `search_ref` to a search, `destination` to a
  destination node, `property` to a property.
- A **search card** points through to the full **property**.
- A **property** reaches its **reviews** through `reviews_ref` and the city it
  sits in through `destination_ref`.
- A **review** points back to its **property**.
- A **destination** climbs to its parent through `parent_ref`, descends to its
  children through `children_ref`, lists its hotels through `properties_ref`, and
  opens a search through `search_ref`.
- A **sitemap index** points at its seeds through `seeds_ref`, and a **seed** fans
  into a `destination` or a `property`. Because the indexes are listed in
  robots.txt, a crawl can start from `booking://sitemaps` with no prior id.

The geographic tree and the property graph connect through the city node, so a
crawl from any seed reaches the reachable public estate.

## Walking the graph

`ls` lists the members of a collection, and every member is itself an addressable
URI, so a host can follow the graph and write it to disk:

```bash
ant ls     booking://destinations/country/us         # the country's child nodes
ant export booking://properties/city/us/orlando --follow 1 --to ./data
```

Because edges between records carry their target type, `ant export --follow` and
`ant graph` walk those edges across tools when a link points at another site's
scheme.

## Two tiers, same as the CLI

The driver reads the same two tiers the command does. `destination`,
`destinations`, and `properties` are the reliable destination estate and read
from anywhere. `property`, `reviews`, and `search` are the best-effort
interactive client, which can hit the bot wall from a datacenter. A host sees
that as the same need-auth outcome the CLI reports as exit code 4.

## Why this is the same code

The driver and the binary share one definition per operation. A resolver op
answers both `booking property` on the command line and
`ant get booking://property/...` through a host, from the same handler and the
same client. There is no second implementation to keep in step.
