package booking

import "time"

// Config is the resolved settings a Client reads. domain.go's ClientFromConfig
// maps the framework's kit.Config onto this, so the standalone binary and a host
// pace, identify, and locate themselves the same way. There are no credential
// fields: Booking exposes no free public API key, so every surface is anonymous.
type Config struct {
	UserAgent string
	Delay     time.Duration // minimum gap between requests
	Retries   int           // retries on 429/5xx
	Timeout   time.Duration // per-request timeout

	Locale   string // language/locale for names, dates, prices, e.g. "en-us"
	Currency string // ISO 4217 price currency, e.g. "USD"

	// Search occupancy and dates. A price is filled only when CheckIn and
	// CheckOut are both set; otherwise the price fields stay empty.
	CheckIn  string // YYYY-MM-DD
	CheckOut string // YYYY-MM-DD
	Adults   int
	Children int
	Rooms    int

	CacheDir string
	NoCache  bool
	CacheTTL time.Duration
	Refresh  bool // refetch and rewrite the cache, ignoring any hit
}

// DefaultUserAgent identifies the client to Booking.com. It is honest: it names
// the tool rather than impersonating a browser or a specific search crawler.
const DefaultUserAgent = "booking-cli/0.1 (+https://github.com/tamnd/booking-cli)"

// DefaultCacheTTL is how long a cached response stays fresh by default.
const DefaultCacheTTL = 24 * time.Hour

// DefaultConfig returns the baseline settings.
func DefaultConfig() Config {
	return Config{
		UserAgent: DefaultUserAgent,
		Delay:     2 * time.Second,
		Retries:   3,
		Timeout:   30 * time.Second,
		Locale:    "en-us",
		Currency:  "USD",
		Adults:    2,
		Children:  0,
		Rooms:     1,
		CacheTTL:  DefaultCacheTTL,
	}
}
