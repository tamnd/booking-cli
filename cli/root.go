// Package cli assembles the booking command tree from the booking
// domain on top of the any-cli/kit framework.
package cli

import (
	"strconv"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/booking-cli/booking"
)

// Build metadata, set via -ldflags at release time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// builder holds the domain-global flags while the app is assembled, then folds
// them onto the resolved config in finalize, using the exact keys
// ClientFromConfig reads.
type builder struct {
	userAgent string
	locale    string
	currency  string
	checkin   string
	checkout  string
	adults    int
	children  int
	rooms     int
	cacheTTL  string
	refresh   bool
}

// NewApp assembles the kit application from the booking domain. The domain's
// Register installs the client factory and every operation, so the binary and a
// host (ant, which blank-imports the package) share one source of truth. This
// package adds the domain-global flags and the version command; kit.Run turns the
// App into the CLI, plus the serve and mcp surfaces and the typed-error-to-exit-
// code mapping.
//
// To add a command, declare it in booking/domain.go with kit.Handle and it
// appears here automatically. Reach for app.AddCommand only for a verb that does
// not fit the emit-records shape, the way version does below.
func NewApp() *kit.App {
	b := &builder{}
	id := booking.Identity()
	id.Version = Version

	app := kit.New(id, kit.WithDefaults(booking.Defaults))
	app.GlobalFlags(b.globals)
	app.Finalize(b.finalize)

	booking.Domain{}.Register(app)
	app.AddCommand(newVersionCmd())
	return app
}

func (b *builder) globals(f *kit.FlagSet) {
	def := booking.DefaultConfig()
	f.StringVar(&b.userAgent, "user-agent", booking.DefaultUserAgent, "User-Agent sent with each request")
	f.StringVar(&b.locale, "locale", def.Locale, "language and locale for names, dates, and prices")
	f.StringVar(&b.currency, "currency", def.Currency, "ISO 4217 currency for prices")
	f.StringVar(&b.checkin, "checkin", "", "check-in date YYYY-MM-DD (a price is filled only with both dates)")
	f.StringVar(&b.checkout, "checkout", "", "check-out date YYYY-MM-DD")
	f.IntVar(&b.adults, "adults", def.Adults, "number of adults for occupancy and pricing")
	f.IntVar(&b.children, "children", def.Children, "number of children for occupancy")
	f.IntVar(&b.rooms, "rooms", def.Rooms, "number of rooms for occupancy")
	f.StringVar(&b.cacheTTL, "cache-ttl", booking.DefaultCacheTTL.String(), "how long a cached response stays fresh")
	f.BoolVar(&b.refresh, "refresh", false, "fetch fresh copies and rewrite the cache, ignoring any hit")
}

func (b *builder) finalize(c *kit.Config) {
	if c.Extra == nil {
		c.Extra = map[string]string{}
	}
	set := func(k, v string) {
		if v != "" {
			c.Extra[k] = v
		}
	}
	set("user-agent", b.userAgent)
	set("locale", b.locale)
	set("currency", b.currency)
	set("checkin", b.checkin)
	set("checkout", b.checkout)
	set("cache-ttl", b.cacheTTL)
	c.Extra["adults"] = strconv.Itoa(b.adults)
	c.Extra["children"] = strconv.Itoa(b.children)
	c.Extra["rooms"] = strconv.Itoa(b.rooms)
	if b.refresh {
		c.Extra["refresh"] = "true"
	}
}
