package booking

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// estate.go parses Booking.com's destination estate: the country, region, city,
// district, landmark, and airport landing pages that exist to be crawled. These
// are the reliable tier and read from anywhere. The parser keys on the stable URL
// patterns of the links rather than on CSS class names, so it tolerates layout
// drift. The three operations share it: destination returns the node, destinations
// returns its child nodes, and properties returns the property cards.

var (
	anchorRE  = regexp.MustCompile(`(?is)<a\b[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	tagStrip  = regexp.MustCompile(`(?s)<[^>]+>`)
	h1RE      = regexp.MustCompile(`(?is)<h1[^>]*>(.*?)</h1>`)
	titleRE   = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	countRE   = regexp.MustCompile(`([\d][\d.,]*)\s*(?:propert|hotel|accommodation)`)
	numInText = regexp.MustCompile(`([\d][\d.,]*)`)
)

// GetDestination fetches one landing page and returns its node.
func (c *Client) GetDestination(ctx context.Context, ref string) (*Destination, error) {
	kind, cc, slug, ok := parseDestRef(ref)
	if !ok {
		return nil, ErrUsage
	}
	id := destID(kind, cc, slug)
	pageURL := URLFor("destination", id)
	body, err := c.get(ctx, pageURL)
	if err != nil {
		return nil, err
	}
	d := &Destination{
		ID:            id,
		Kind:          kind,
		Country:       cc,
		Name:          destName(body, slug, cc),
		PropertyCount: headlineCount(body),
		URL:           pageURL,
		ChildrenRef:   id,
		PropertiesRef: id,
	}
	d.Lat, d.Lng = landingGeo(body, slug)
	d.SearchRef = d.Name
	d.ParentRef = parentRef(body, kind, cc)
	return d, nil
}

// ListChildren returns the destination nodes linked from a landing page. A
// country page links down to its regions and cities and also across to peer
// countries, so this returns both: the extra peer edges are what let a crawl
// reach every country from any one of them. Each node's ParentRef is set by its
// own place in the tree rather than to this page, so a peer country is not given a
// false parent.
func (c *Client) ListChildren(ctx context.Context, ref string, limit int) ([]*Destination, error) {
	selfKind, selfCC, selfSlug, ok := parseDestRef(ref)
	if !ok {
		return nil, ErrUsage
	}
	selfID := destID(selfKind, selfCC, selfSlug)
	body, err := c.get(ctx, URLFor("destination", selfID))
	if err != nil {
		return nil, err
	}
	var out []*Destination
	seen := map[string]bool{}
	for _, a := range anchorRE.FindAllSubmatch(body, -1) {
		r := Classify(string(a[1]))
		if r.Kind != "destination" || r.ID == selfID || seen[r.ID] {
			continue
		}
		kind, cc, slug, _ := parseDestRef(r.ID)
		name := anchorText(a[2])
		if name == "" {
			name = nameFromSlug(slug)
		}
		seen[r.ID] = true
		d := &Destination{
			ID:            r.ID,
			Kind:          kind,
			Country:       cc,
			Name:          name,
			PropertyCount: numFrom(anchorText(a[2])),
			URL:           URLFor("destination", r.ID),
			ParentRef:     structuralParent(kind, cc, selfID),
			ChildrenRef:   r.ID,
			PropertiesRef: r.ID,
			SearchRef:     name,
		}
		out = append(out, d)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

// structuralParent returns the parent id a child node should carry given its own
// kind, rather than assuming the page it was found on is its parent. A region's
// parent is its country, a city's is its country, and a deeper node defaults to
// the page it came from when that page is a plausible ancestor. A country has no
// parent.
func structuralParent(childKind, childCC, fromID string) string {
	switch childKind {
	case "country":
		return ""
	case "region", "city":
		return "country/" + childCC
	case "district", "landmark", "airport":
		if fromKind, _, _, ok := splitDestID(fromID); ok && fromKind == "city" {
			return fromID
		}
		return "country/" + childCC
	default:
		return "country/" + childCC
	}
}

// ListProperties returns the property cards linked from a landing page.
func (c *Client) ListProperties(ctx context.Context, ref string, limit int) ([]*Property, error) {
	kind, cc, slug, ok := parseDestRef(ref)
	if !ok {
		return nil, ErrUsage
	}
	selfID := destID(kind, cc, slug)
	body, err := c.get(ctx, URLFor("destination", selfID))
	if err != nil {
		return nil, err
	}
	cards := propertyCards(body)
	if kind == "city" || kind == "district" {
		for _, p := range cards {
			p.DestinationRef = selfID
		}
	}
	return limitCards(cards, limit), nil
}

// imgSrcRE pulls the source from a card's <img>, accepting the lazy-load
// data-src as well as src, since Booking defers the real image to data-src.
var imgSrcRE = regexp.MustCompile(`(?is)<img\b[^>]*?\b(?:data-src|src)="([^"]+)"`)

// propertyCards returns the property cards on a results or landing page, deduped
// by id and in page order. Booking renders each card as two anchors to the same
// /hotel/ link: one wraps the thumbnail image, the other the name. This merges
// them by id so a card carries both the name and the thumbnail.
func propertyCards(body []byte) []*Property {
	var out []*Property
	idx := map[string]*Property{}
	for _, a := range anchorRE.FindAllSubmatch(body, -1) {
		r := Classify(string(a[1]))
		if r.Kind != "property" {
			continue
		}
		p := idx[r.ID]
		if p == nil {
			p = &Property{ID: r.ID, URL: URLFor("property", r.ID), ReviewsRef: r.ID}
			idx[r.ID] = p
			out = append(out, p)
		}
		if p.Name == "" {
			if name := anchorText(a[2]); name != "" {
				p.Name = name
			}
		}
		if p.Image == "" {
			if img := cardImage(a[2]); img != "" {
				p.Image = img
			}
		}
	}
	return out
}

// cardImage returns the first hotel thumbnail in a card anchor's inner HTML.
func cardImage(inner []byte) string {
	for _, m := range imgSrcRE.FindAllSubmatch(inner, -1) {
		src := squish(string(m[1]))
		if strings.Contains(src, "/images/hotel/") || strings.Contains(src, "bstatic.com") {
			return src
		}
	}
	return ""
}

// limitCards trims a card slice to limit, keeping all when limit is zero.
func limitCards(cards []*Property, limit int) []*Property {
	if limit > 0 && len(cards) > limit {
		return cards[:limit]
	}
	return cards
}

// parseDestRef resolves a destination reference (a URL or a bare
// "<kind>/<cc>[/<slug>]") to its parts.
func parseDestRef(ref string) (kind, cc, slug string, ok bool) {
	r := Classify(ref)
	if r.Kind != "destination" {
		return "", "", "", false
	}
	return splitDestID(r.ID)
}

// splitDestID splits "<kind>/<cc>[/<slug>]" into its parts.
func splitDestID(id string) (kind, cc, slug string, ok bool) {
	segs := splitPath(id)
	switch len(segs) {
	case 2:
		return segs[0], segs[1], "", true
	case 3:
		return segs[0], segs[1], segs[2], true
	default:
		return "", "", "", false
	}
}

// destID rebuilds the canonical id from parts.
func destID(kind, cc, slug string) string {
	if slug == "" {
		return kind + "/" + cc
	}
	return kind + "/" + cc + "/" + slug
}

// parentRef returns the parent node id for a kind, reading a real ancestor link
// from the body where the hierarchy needs it.
func parentRef(body []byte, kind, cc string) string {
	switch kind {
	case "country":
		return ""
	case "region":
		return "country/" + cc
	case "city":
		if id := firstDestLink(body, "region", cc); id != "" {
			return id
		}
		return "country/" + cc
	case "district", "landmark", "airport":
		if id := firstDestLink(body, "city", cc); id != "" {
			return id
		}
		return "country/" + cc
	default:
		return "country/" + cc
	}
}

// firstDestLink returns the id of the first link of a given kind and country code
// found in the body.
func firstDestLink(body []byte, kind, cc string) string {
	re := regexp.MustCompile(`/` + kind + `/` + cc + `/([^"./?]+)\.html`)
	if m := re.FindSubmatch(body); m != nil {
		return kind + "/" + cc + "/" + string(m[1])
	}
	return ""
}

// landingGeo reads the node's coordinates from a JSON-LD object on the landing
// page. City pages carry a City object with a geo block, and some place pages a
// Place object, so this walks the islands for an object that has geo, preferring
// one whose @id names the page's slug, then a City, then any. It returns zeroes
// when the page carries none, which is the common case for country and region
// pages.
func landingGeo(body []byte, slug string) (lat, lng float64) {
	type best struct {
		lat, lng float64
		score    int
	}
	var pick best
	var walk func(node any)
	walk = func(node any) {
		switch n := node.(type) {
		case map[string]any:
			if g, ok := n["geo"].(map[string]any); ok {
				la, okLa := geoNum(g["latitude"])
				ln, okLn := geoNum(g["longitude"])
				if okLa && okLn {
					score := 1
					if t, _ := n["@type"].(string); strings.EqualFold(t, "City") {
						score = 2
					}
					if id, _ := n["@id"].(string); slug != "" && strings.Contains(strings.ToLower(id), strings.ToLower(slug)) {
						score = 3
					}
					if score > pick.score {
						pick = best{la, ln, score}
					}
				}
			}
			for _, v := range n {
				walk(v)
			}
		case []any:
			for _, v := range n {
				walk(v)
			}
		}
	}
	for _, raw := range ldBlocks(body) {
		var doc any
		if err := json.Unmarshal(raw, &doc); err != nil {
			continue
		}
		walk(doc)
	}
	return pick.lat, pick.lng
}

// geoNum reads a coordinate that JSON-LD encodes as a number or a quoted number.
func geoNum(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

// destName reads the node's display name from the page heading or title, falling
// back to the slug or the country code.
func destName(body []byte, slug, cc string) string {
	if m := h1RE.FindSubmatch(body); m != nil {
		if name := anchorText(m[1]); name != "" {
			return name
		}
	}
	if m := titleRE.FindSubmatch(body); m != nil {
		if name := anchorText(m[1]); name != "" {
			return firstClause(name)
		}
	}
	if slug != "" {
		return nameFromSlug(slug)
	}
	return strings.ToUpper(cc)
}

// firstClause keeps the part of a title before a separator, so "Hotels in Orlando
// | Booking.com" reduces to a short name.
func firstClause(s string) string {
	for _, sep := range []string{" | ", " - ", ": "} {
		if i := strings.Index(s, sep); i > 0 {
			return strings.TrimSpace(s[:i])
		}
	}
	return s
}

// headlineCount pulls the property count out of the page's headline text.
func headlineCount(body []byte) int {
	if m := countRE.FindSubmatch(body); m != nil {
		return parseCount(string(m[1]))
	}
	return 0
}

// numFrom pulls the first number out of a string, for a count next to a link.
func numFrom(s string) int {
	if m := numInText.FindString(s); m != "" {
		return parseCount(m)
	}
	return 0
}

// parseCount turns "85,215" or "1.234" grouping into an int.
func parseCount(s string) int {
	s = strings.NewReplacer(",", "", ".", "", " ", "").Replace(s)
	n, _ := strconv.Atoi(s)
	return n
}

// anchorText strips tags and collapses whitespace from anchor inner HTML.
func anchorText(html []byte) string {
	return squish(tagStrip.ReplaceAllString(string(html), " "))
}

// nameFromSlug turns "soho-london" into "Soho London".
func nameFromSlug(slug string) string {
	parts := strings.Split(slug, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}
