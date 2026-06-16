package booking

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes booking as a kit Domain: a driver that a multi-domain host
// (ant) enables with a single blank import,
//
//	import _ "github.com/tamnd/booking-cli/booking"
//
// exactly as a database/sql program enables a driver with `import _
// "github.com/lib/pq"`. The init below registers it; the host then dereferences
// booking:// URIs by routing to the operations Register installs. The same Domain
// also builds the standalone booking binary (see cli.NewApp), so the binary and a
// host share one source of truth.
func init() { kit.Register(Domain{}) }

// Domain is the booking driver. It carries no state; the per-run client is built
// by the factory Register hands kit.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against, and
// the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme:   "booking",
		Hosts:    []string{Host, "booking.com"},
		Identity: Identity(),
	}
}

// Identity is the fixed description of the booking CLI, shared by the domain and
// the standalone composition root so help and version read the same everywhere.
func Identity() kit.Identity {
	return kit.Identity{
		Binary: "booking",
		Short:  "Read public Booking.com destinations, properties, reviews, and suggestions into structured records",
		Long: `booking reads public Booking.com data over plain HTTPS in one web
plane with two reliability tiers. The destination estate (country,
region, city, district, landmark, and airport landing pages) is built to
be crawled and reads from anywhere, and is the home of destination,
destinations, and properties. The interactive client (the property page,
reviews, search, and autocomplete) is fronted by a bot manager and is
best-effort from a datacenter, where a wall returns exit 4; it works from
a residential or mobile connection. There is no API key: every surface is
anonymous. A nightly price is filled only when --checkin and --checkout
are set, never invented. booking returns records as a table, JSON, JSONL,
CSV, TSV, or URLs, and serves the same operations over HTTP and MCP.

booking is an independent tool and is not affiliated with Booking.com.`,
		Site: BaseURL,
		Repo: "https://github.com/tamnd/booking-cli",
	}
}

// Register installs the client factory and every operation onto app. A resolver
// op (Single) names its own record type and answers `ant get`; a List op
// enumerates a parent resource's members and answers `ant ls`. Each list op names
// its own collection authority, distinct from the property and destination
// resolvers, so booking://search/<term>, booking://reviews/<id>,
// booking://destinations/<ref>, booking://properties/<ref>, and
// booking://suggest/<prefix> each reach the right op rather than shadowing one
// another.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)
	app.CommandGroup("read", "Read public Booking.com data")
	app.CommandGroup("ref", "Resolve references to ids and URLs (offline)")

	kit.Handle(app, kit.OpMeta{
		Name: "search", Group: "read", List: true,
		Summary: "Search properties by free-text destination (best-effort tier)",
		URIType: "search",
		Args:    []kit.Arg{{Name: "destination", Help: "a place to search, e.g. \"Orlando\""}},
	}, search)

	kit.Handle(app, kit.OpMeta{
		Name: "property", Group: "read", Single: true,
		Summary: "Show one property by id or URL (best-effort tier)",
		URIType: "property", Resolver: true,
		Args: []kit.Arg{{Name: "ref", Help: "property id \"<cc>/<slug>\" or /hotel/ URL"}},
	}, getProperty)

	kit.Handle(app, kit.OpMeta{
		Name: "reviews", Group: "read", List: true,
		Summary: "List a property's reviews (best-effort tier)",
		URIType: "reviews",
		Args:    []kit.Arg{{Name: "ref", Help: "property id \"<cc>/<slug>\" or /hotel/ URL"}},
	}, reviews)

	kit.Handle(app, kit.OpMeta{
		Name: "destination", Group: "read", Single: true,
		Summary: "Show one destination node (country/region/city/district/landmark/airport)",
		URIType: "destination", Resolver: true,
		Args: []kit.Arg{{Name: "ref", Help: "destination ref \"<kind>/<cc>[/<slug>]\" or landing URL"}},
	}, getDestination)

	kit.Handle(app, kit.OpMeta{
		Name: "destinations", Group: "read", List: true,
		Summary: "List a destination's child nodes (walks the taxonomy down)",
		URIType: "destinations",
		Args:    []kit.Arg{{Name: "ref", Help: "destination ref or landing URL"}},
	}, listDestinations)

	kit.Handle(app, kit.OpMeta{
		Name: "properties", Group: "read", List: true,
		Summary: "List the properties on a destination landing page (reliable tier)",
		URIType: "properties",
		Args:    []kit.Arg{{Name: "ref", Help: "destination ref or landing URL"}},
	}, listProperties)

	kit.Handle(app, kit.OpMeta{
		Name: "suggest", Group: "read", List: true,
		Summary: "Autocomplete destinations and properties for a prefix",
		URIType: "suggest",
		Args:    []kit.Arg{{Name: "prefix", Help: "the typed prefix"}},
	}, suggest)

	kit.Handle(app, kit.OpMeta{
		Name: "sitemap", Group: "read", List: true,
		Summary: "Enumerate a kind's landing pages from Booking's sitemaps (the crawl root)",
		URIType: "sitemap",
		Args:    []kit.Arg{{Name: "kind", Help: "country, region, city, district, landmark, airport, or hotel"}},
	}, sitemap)

	// Reference tools (offline).
	kit.Handle(app, kit.OpMeta{
		Name: "id", Parent: "ref", Single: true,
		Summary: "Classify a reference into its (kind, id)",
		Args:    []kit.Arg{{Name: "ref", Help: "any Booking.com URL, path, or id"}},
	}, classifyRef)

	kit.Handle(app, kit.OpMeta{
		Name: "url", Parent: "ref", Single: true,
		Summary: "Build the canonical URL for a (kind, id)",
		Args: []kit.Arg{
			{Name: "kind", Help: "property, destination, or search"},
			{Name: "id", Help: "the id for that kind"},
		},
	}, buildURL)
}

// newClient builds the client from the host-resolved config, so a host and the
// standalone binary pace and identify themselves the same way.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	return ClientFromConfig(cfg), nil
}

// ClientFromConfig maps the framework config onto a booking.Config and returns a
// client. There are no credential keys: Booking offers no free public API, so
// every surface is anonymous.
func ClientFromConfig(cfg kit.Config) *Client {
	bc := DefaultConfig()
	if cfg.Rate > 0 {
		bc.Delay = cfg.Rate
	}
	if cfg.Retries >= 0 {
		bc.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		bc.Timeout = cfg.Timeout
	}
	if ua := cfg.Extra["user-agent"]; ua != "" {
		bc.UserAgent = ua
	} else if cfg.UserAgent != "" {
		bc.UserAgent = cfg.UserAgent
	}
	if v := cfg.Extra["locale"]; v != "" {
		bc.Locale = v
	}
	if v := cfg.Extra["currency"]; v != "" {
		bc.Currency = v
	}
	if v := cfg.Extra["checkin"]; v != "" {
		bc.CheckIn = v
	}
	if v := cfg.Extra["checkout"]; v != "" {
		bc.CheckOut = v
	}
	if v := cfg.Extra["adults"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			bc.Adults = n
		}
	}
	if v := cfg.Extra["children"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			bc.Children = n
		}
	}
	if v := cfg.Extra["rooms"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			bc.Rooms = n
		}
	}
	bc.CacheDir = cfg.CacheDir
	bc.NoCache = cfg.NoCache
	if ttl := cfg.Extra["cache-ttl"]; ttl != "" {
		if d, err := time.ParseDuration(ttl); err == nil {
			bc.CacheTTL = d
		}
	}
	bc.Refresh = cfg.Extra["refresh"] == "true"
	return NewClient(bc)
}

// Defaults seeds the framework baseline with booking's own values, so an unset
// --rate or --timeout uses the booking default rather than the generic kit one.
func Defaults(c *kit.Config) {
	def := DefaultConfig()
	c.Rate = def.Delay
	c.Retries = def.Retries
	c.Timeout = def.Timeout
	c.UserAgent = def.UserAgent
}

// Classify turns any accepted input into the canonical (type, id), so `ant
// resolve` and `ant url` touch no network.
func (Domain) Classify(input string) (uriType, id string, err error) {
	r := Classify(input)
	if r.Kind == "unknown" {
		return "", "", errs.Usage("unrecognized booking reference: %q", input)
	}
	return r.Kind, r.ID, nil
}

// Locate is the inverse: the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	u := URLFor(uriType, id)
	if u == "" {
		return "", errs.Usage("booking has no resource type %q", uriType)
	}
	return u, nil
}

// mapErr translates a library error into a kit error so the exit code matches the
// rest of the fleet: a missing entity reads as not found (exit 6), a throttle as
// rate limited (exit 5), the bot wall as need-auth (exit 4), and a caught bad
// argument as usage (exit 2).
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ErrNotFound):
		return errs.NotFound("%s", err.Error())
	case errors.Is(err, ErrRateLimited):
		return errs.RateLimited("%s", err.Error())
	case errors.Is(err, ErrBlocked):
		return errs.NeedAuth("%s", err.Error())
	case errors.Is(err, ErrUsage):
		return errs.Usage("%s", err.Error())
	default:
		return err
	}
}
