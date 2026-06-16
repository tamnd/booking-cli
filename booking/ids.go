package booking

import (
	"net/url"
	"regexp"
	"strings"
)

// ids.go is the offline reference layer: Classify turns any Booking.com URL or
// bare reference into a canonical (kind, id), and URLFor builds the canonical URL
// for a (kind, id). Both are pure and never touch the network, so `booking ref
// id` and `booking ref url` (and a host's resolve/url) answer instantly.
//
// The kinds:
//   - property:    a hotel page, id "<cc>/<slug>"             /hotel/<cc>/<slug>.html
//   - destination: a node of the geographic tree, id "<kind>/<cc>[/<slug>]"
//   - search:      a free-text destination search, id the ss term
//
// reviews, destinations, and properties are list authorities derived from a
// property or destination id, not separate URL kinds, but URLFor still answers
// them so a host can build a canonical page URL for any edge.

// destKinds are the first path segment of a destination landing page.
var destKinds = map[string]bool{
	"country": true, "region": true, "city": true,
	"district": true, "landmark": true, "airport": true,
}

// localeSuffix matches a trailing ".<locale>" on a hotel slug, e.g. the "en-gb"
// in "the-savoy.en-gb". Locales are two letters optionally followed by a region.
var localeSuffix = regexp.MustCompile(`\.[a-z]{2}(-[a-z]+)?$`)

// Classify resolves a reference offline. It accepts a full Booking.com URL, a
// path, or a bare id ("gb/the-savoy", "city/us/orlando").
func Classify(input string) Ref {
	in := strings.TrimSpace(input)
	r := Ref{Input: input, Kind: "unknown"}

	path := in
	var query url.Values
	if u, err := url.Parse(in); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		path = u.Path
		query = u.Query()
	}
	path = strings.Trim(path, "/")

	// /searchresults.html?ss=<term>
	if strings.HasPrefix(path, "searchresults") {
		if query != nil {
			if ss := strings.TrimSpace(query.Get("ss")); ss != "" {
				r.Kind, r.ID = "search", ss
				r.URL = URLFor(r.Kind, r.ID)
				return r
			}
		}
		return r
	}

	segs := splitPath(path)
	if len(segs) == 0 {
		return r
	}

	switch {
	case segs[0] == "hotel" && len(segs) >= 3:
		// /hotel/<cc>/<slug>.<locale>.html
		cc := segs[1]
		slug := stripPageSuffix(segs[2])
		if cc != "" && slug != "" {
			r.Kind, r.ID = "property", cc+"/"+slug
		}
	case segs[0] == "country" && len(segs) >= 2:
		cc := stripPageSuffix(segs[1])
		if cc != "" {
			r.Kind, r.ID = "destination", "country/"+cc
		}
	case destKinds[segs[0]] && len(segs) >= 3:
		// /<kind>/<cc>/<slug>.html
		cc := segs[1]
		slug := stripPageSuffix(segs[2])
		if cc != "" && slug != "" {
			r.Kind, r.ID = "destination", segs[0]+"/"+cc+"/"+slug
		}
	case destKinds[segs[0]] && len(segs) == 2 && segs[0] == "country":
		// handled above; here for completeness
	case len(segs) == 2 && len(segs[0]) == 2:
		// bare "<cc>/<slug>"
		r.Kind, r.ID = "property", segs[0]+"/"+stripPageSuffix(segs[1])
	}

	if r.Kind != "unknown" {
		r.URL = URLFor(r.Kind, r.ID)
	}
	return r
}

// URLFor builds the canonical live URL for a (kind, id), or "" if it cannot.
func URLFor(kind, id string) string {
	id = strings.Trim(id, "/")
	switch kind {
	case "property", "reviews":
		// id is "<cc>/<slug>"; reviews live on the property page
		if id == "" {
			return ""
		}
		return BaseURL + "/hotel/" + id + ".html"
	case "destination", "destinations", "properties":
		if id == "" {
			return ""
		}
		if strings.HasPrefix(id, "country/") {
			return BaseURL + "/country/" + strings.TrimPrefix(id, "country/") + ".html"
		}
		return BaseURL + "/" + id + ".html"
	case "search":
		return BaseURL + "/searchresults.html?ss=" + url.QueryEscape(id)
	default:
		return ""
	}
}

// splitPath splits a cleaned path into its segments, dropping empties.
func splitPath(path string) []string {
	var out []string
	for _, s := range strings.Split(path, "/") {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// stripPageSuffix removes a trailing ".html" and any ".<locale>" before it, so
// "the-savoy.en-gb.html" becomes "the-savoy" and "us.html" becomes "us".
func stripPageSuffix(seg string) string {
	seg = strings.TrimSuffix(seg, ".html")
	seg = localeSuffix.ReplaceAllString(seg, "")
	return seg
}
