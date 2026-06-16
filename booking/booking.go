// Package booking is the library behind the booking command line: the HTTP
// client, the offline reference layer, and the typed records read from public
// Booking.com surfaces.
//
// Booking.com has one web plane with two reliability tiers. The destination
// estate (country, region, city, district, landmark, and airport landing pages)
// is built to be crawled and reads from anywhere; the interactive client (the
// property page, search, reviews, and autocomplete) is fronted by a bot manager
// and is best-effort from a datacenter. The Client below GETs both, paces and
// retries politely, caches on disk, and turns a walled response into ErrBlocked
// before any parser sees it. There is no API key: every surface is anonymous.
package booking

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Host is the site this client talks to, and the host the URI driver in
// domain.go claims.
const Host = "www.booking.com"

// BaseURL is the root every request is built from.
const BaseURL = "https://" + Host

// Client reads public Booking.com data over HTTP.
type Client struct {
	HTTP *http.Client
	cfg  Config
	last time.Time
}

// NewClient returns a Client configured from cfg.
func NewClient(cfg Config) *Client {
	if cfg.UserAgent == "" {
		cfg.UserAgent = DefaultUserAgent
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &Client{
		HTTP: &http.Client{Timeout: cfg.Timeout},
		cfg:  cfg,
	}
}

// get fetches a URL and returns the body. It serves from cache when fresh, paces
// and retries transient failures, and classifies a walled response as ErrBlocked.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	if b := c.cacheGet(rawURL); b != nil {
		return b, nil
	}
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			c.cachePut(rawURL, body)
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	if errors.Is(lastErr, ErrRateLimited) {
		return nil, ErrRateLimited
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) (body []byte, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/json")
	if c.cfg.Locale != "" {
		req.Header.Set("Accept-Language", c.cfg.Locale)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		// A reset or handshake failure mid-request is treated as retryable here;
		// the get loop turns a persistent failure into the wrapped error.
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	switch {
	case resp.StatusCode == http.StatusForbidden:
		return nil, false, ErrBlocked
	case resp.StatusCode == http.StatusAccepted:
		// The bot manager on the interactive tier answers a challenged request
		// with 202 and an interstitial body in place of the real page, so a 202 is
		// the wall, not a partial success.
		return nil, false, ErrBlocked
	case resp.StatusCode == http.StatusNotFound:
		return nil, false, ErrNotFound
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, true, ErrRateLimited
	case resp.StatusCode >= 500:
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	case resp.StatusCode != http.StatusOK:
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	if isChallenge(b) {
		return nil, false, ErrBlocked
	}
	return b, false, nil
}

// pace blocks until at least Delay has passed since the previous request.
func (c *Client) pace() {
	if c.cfg.Delay <= 0 {
		return
	}
	if wait := c.cfg.Delay - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// challengeMarkers are byte signatures of a bot-manager interstitial served with
// a 200 in place of the real page.
var challengeMarkers = [][]byte{
	[]byte("are you a robot"),
	[]byte("are you human"),
	[]byte("captcha-delivery.com"),
	[]byte("px-captcha"),
	[]byte("window._pxappid"),
	[]byte("cf-challenge"),
	[]byte("challenge-platform"),
}

// isChallenge reports whether a 200 body is a bot-manager challenge rather than a
// real page, by looking for a known interstitial marker in the head of the body.
func isChallenge(body []byte) bool {
	head := body
	if len(head) > 8192 {
		head = head[:8192]
	}
	lower := bytes.ToLower(head)
	for _, m := range challengeMarkers {
		if bytes.Contains(lower, m) {
			return true
		}
	}
	return false
}

// squish collapses internal whitespace and trims, for text pulled out of HTML.
func squish(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
