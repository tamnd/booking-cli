package booking

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

// search.go reads the search results page, the most heavily walled surface: search
// is the action the bot manager most wants to gate, so a datacenter request is the
// most likely of all to be challenged. The reliable alternative for "list the
// properties in a place" is ListProperties over a destination landing page, which
// needs no dates and is built to be crawled. Search is what carries free-text
// matching and the dated query.
//
// A nightly price is filled only when both dates are supplied and the card carries
// one; it is never invented. In practice the results HTML carries the cards as
// /hotel/ links, which this parser turns into Property records the property
// operation can then read in full.

// Search returns up to limit property cards for a free-text destination.
func (c *Client) Search(ctx context.Context, dest string, limit int) ([]*Property, error) {
	dest = strings.TrimSpace(dest)
	if dest == "" {
		return nil, ErrUsage
	}
	body, err := c.get(ctx, c.searchURL(dest))
	if err != nil {
		return nil, err
	}
	var out []*Property
	seen := map[string]bool{}
	for _, a := range anchorRE.FindAllSubmatch(body, -1) {
		r := Classify(string(a[1]))
		if r.Kind != "property" || seen[r.ID] {
			continue
		}
		seen[r.ID] = true
		out = append(out, &Property{
			ID:         r.ID,
			Name:       anchorText(a[2]),
			URL:        URLFor("property", r.ID),
			ReviewsRef: r.ID,
		})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

// searchURL builds the searchresults URL with the configured occupancy, currency,
// locale, and (when set) dates.
func (c *Client) searchURL(dest string) string {
	q := url.Values{}
	q.Set("ss", dest)
	if c.cfg.CheckIn != "" {
		q.Set("checkin", c.cfg.CheckIn)
	}
	if c.cfg.CheckOut != "" {
		q.Set("checkout", c.cfg.CheckOut)
	}
	if c.cfg.Adults > 0 {
		q.Set("group_adults", strconv.Itoa(c.cfg.Adults))
	}
	if c.cfg.Children > 0 {
		q.Set("group_children", strconv.Itoa(c.cfg.Children))
	}
	if c.cfg.Rooms > 0 {
		q.Set("no_rooms", strconv.Itoa(c.cfg.Rooms))
	}
	if c.cfg.Currency != "" {
		q.Set("selected_currency", c.cfg.Currency)
	}
	return BaseURL + "/searchresults.html?" + q.Encode()
}
