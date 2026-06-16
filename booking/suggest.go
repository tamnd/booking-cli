package booking

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
)

// suggest.go reads the destination autocomplete endpoint the search box calls. A
// typed prefix becomes a set of destinations and properties, each carrying an edge
// into the rest of the graph, which makes it the natural seed for a crawl. It is
// best-effort: a JSON endpoint can be gated, though it is lighter than search.

// autocompleteHost is the host the search box's autocomplete endpoint lives on.
const autocompleteHost = "https://accommodations.booking.com"

// autocompleteResp is the endpoint's shape: a list of matches, each a place or a
// property. Fields are decoded leniently because the endpoint varies by locale.
type autocompleteResp struct {
	Results []autocompleteMatch `json:"results"`
}

type autocompleteMatch struct {
	Label     string `json:"label"`
	Type      string `json:"type"`      // city, region, district, country, landmark, airport, hotel
	DestType  string `json:"dest_type"` // same vocabulary, when present
	DestID    string `json:"dest_id"`
	Country   string `json:"country"`
	CC        string `json:"cc1"`
	Latitude  fnum   `json:"latitude"`
	Longitude fnum   `json:"longitude"`
	NrHotels  int    `json:"nr_hotels"`
	HotelSlug string `json:"hotel_slug"`
	URL       string `json:"url"`
	Roundtrip string `json:"roundtrip"`
}

// Suggest returns up to limit autocomplete matches for a prefix.
func (c *Client) Suggest(ctx context.Context, prefix string, limit int) ([]*Suggestion, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return nil, ErrUsage
	}
	q := url.Values{}
	q.Set("term", prefix)
	if c.cfg.Locale != "" {
		q.Set("language", c.cfg.Locale)
	}
	size := limit
	if size <= 0 {
		size = 10
	}
	q.Set("size", strconv.Itoa(size))
	body, err := c.get(ctx, autocompleteHost+"/autocomplete.json?"+q.Encode())
	if err != nil {
		return nil, err
	}
	var resp autocompleteResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, ErrBlocked // a non-JSON body here is a gated response
	}
	var out []*Suggestion
	for _, m := range resp.Results {
		s := mapSuggestion(prefix, m)
		if s == nil {
			continue
		}
		out = append(out, s)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

// mapSuggestion maps one autocomplete match onto a Suggestion, filling the edge
// that matches its kind. SearchRef is always filled so a prefix can fan into a
// search. A destination or property edge is filled only when the match carries a
// landing URL or a hotel slug a real link can be built from, never a guessed slug.
func mapSuggestion(prefix string, m autocompleteMatch) *Suggestion {
	label := squish(m.Label)
	if label == "" {
		return nil
	}
	kind := m.Type
	if kind == "" {
		kind = m.DestType
	}
	s := &Suggestion{
		Query:         prefix,
		Text:          label,
		Kind:          kind,
		Country:       squish(m.Country),
		PropertyCount: m.NrHotels,
		DestID:        m.DestID,
		DestType:      m.DestType,
		Lat:           m.Latitude.float(),
		Lng:           m.Longitude.float(),
		SearchRef:     label,
	}
	// A landing or hotel URL classifies into a real edge.
	if m.URL != "" {
		if r := Classify(m.URL); r.Kind == "property" {
			s.Property = r.ID
		} else if r.Kind == "destination" {
			s.Destination = r.ID
		}
	}
	if s.Property == "" && kind == "hotel" && m.HotelSlug != "" && m.CC != "" {
		s.Property = m.CC + "/" + m.HotelSlug
	}
	return s
}
