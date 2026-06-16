package booking

import "errors"

// Sentinel errors the library returns; domain.go's mapErr turns each into the
// kit error kind that carries the right exit code (see the spec section 4.4).
var (
	// ErrBlocked is the bot wall: a 403, a connection reset, a challenge or
	// interstitial body, or a gated JSON envelope. It maps to need-auth (exit 4),
	// with a remedy of retrying from a residential or mobile connection or reading
	// the destination estate, which is not gated.
	ErrBlocked = errors.New("blocked by Booking.com's bot wall: retry from a residential or mobile connection, or read the destination estate (country/region/city pages), which is not gated")

	// ErrNotFound is a missing place or property (a 404). Maps to exit 6.
	ErrNotFound = errors.New("not found")

	// ErrRateLimited is a sustained 429 after retries. Maps to exit 5.
	ErrRateLimited = errors.New("rate limited by Booking.com: slow down with --rate or try again later")

	// ErrUsage is a bad argument that the library catches (a malformed date, an
	// unrecognized reference). Maps to exit 2.
	ErrUsage = errors.New("usage")
)
