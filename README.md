# booking

A command line for Booking.com.

`booking` is a single pure-Go binary. It reads public Booking.com data over plain
HTTPS, shapes it into clean records, and prints output that pipes into the rest of
your tools. There is no API key: every surface is anonymous.

The same package is also a [resource-URI driver](#use-it-as-a-resource-uri-driver),
so a host program like [ant](https://github.com/tamnd/ant) can address Booking.com
as `booking://` URIs.

## One web plane, two reliability tiers

Booking.com has no free public API, so `booking` reads the public website. That
website has two tiers, and `booking` is honest about which is which:

- **The destination estate** is the country, region, city, district, landmark, and
  airport landing pages. These exist to be crawled and read from anywhere. They are
  the reliable backbone and the home of `destination`, `destinations`,
  `properties`, and `sitemap`, which reads Booking's own sitemaps as the crawl
  root.
- **The interactive client** is the property page, `reviews`, `search`, and
  `suggest`. These sit behind a bot manager. They work from a residential or mobile
  connection, and are best-effort from a datacenter, where a wall returns exit 4.

A nightly price is filled only when you pass both `--checkin` and `--checkout`. It
is never invented.

## Install

```bash
go install github.com/tamnd/booking-cli/cmd/booking@latest
```

Or grab a prebuilt binary from the [releases](https://github.com/tamnd/booking-cli/releases), or run
the container image:

```bash
docker run --rm ghcr.io/tamnd/booking:latest --help
```

## Usage

```bash
booking sitemap country                       # every country page, the crawl root
booking suggest orlando                       # autocomplete places and hotels
booking destination country/us                # one node of the geographic tree
booking destinations country/us               # its child nodes
booking properties city/us/orlando            # the hotels on a city landing page
booking property gb/the-savoy                 # one property in full (JSON-LD)
booking reviews gb/the-savoy                  # that property's reviews
booking search Orlando --checkin 2026-07-01 --checkout 2026-07-04
booking ref id https://www.booking.com/hotel/gb/the-savoy.html
booking --help                                # the whole command tree
```

Every command shares one output contract:
`-o table|markdown|json|jsonl|csv|tsv|url|raw`, `--fields` to pick columns,
`--template` for a custom line, and `-n` to limit. The default adapts to where
output goes (a color-aware table on a terminal, JSONL in a pipe), so the same
command reads well by hand and parses cleanly downstream.

Records connect into one graph. A sitemap seed fans into a destination or a
property; a suggestion fans into a search, a place, or a hotel; a search card
walks through to the full property; a property reaches its reviews and the city it
sits in; a destination climbs to its parent, descends to its children, and lists
its properties. Starting from `sitemap`, which needs no prior id, a host can crawl
the reachable public estate by following those edges.

## Serve it

The same operations are available over HTTP and as an MCP tool set for agents,
with no extra code:

```bash
booking serve --addr :7777    # GET /v1/property/<id> returns NDJSON
booking mcp                   # speak MCP over stdio
```

## Use it as a resource-URI driver

`booking` registers a `booking` domain the way a program registers a database
driver with `database/sql`. A host enables it with one blank import:

```go
import _ "github.com/tamnd/booking-cli/booking"
```

Then [ant](https://github.com/tamnd/ant) (or any program that links the package)
dereferences `booking://` URIs without knowing anything about Booking.com:

```bash
ant get booking://property/gb/the-savoy      # fetch the record
ant cat booking://property/gb/the-savoy      # just the description
ant ls  booking://properties/city/us/orlando # the hotels on a city page
ant url booking://destination/country/us     # the live https URL
```

## Development

```
cmd/booking/   thin main: hands cli.NewApp to kit.Run
cli/           assembles the kit App from the booking domain
booking/       the library: HTTP client, parsers, data models, and domain.go (the driver)
docs/          tago documentation site
```

```bash
make build      # ./bin/booking
make test       # go test ./...
make vet        # go vet ./...
```

## Releasing

Push a version tag and GitHub Actions runs GoReleaser, which builds the
archives, Linux packages, the multi-arch GHCR image, checksums, SBOMs, and a
cosign signature:

```bash
git tag v0.1.0
git push --tags
```

The Homebrew and Scoop steps self-disable until their tokens exist, so the first
release works with no extra secrets.

## License

booking is an independent tool and is not affiliated with Booking.com. Apache-2.0.
See [LICENSE](LICENSE).
