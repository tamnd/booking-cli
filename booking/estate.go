package booking

import (
	"context"
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
	d.SearchRef = d.Name
	d.ParentRef = parentRef(body, kind, cc)
	return d, nil
}

// ListChildren returns the child destination nodes linked from a landing page.
func (c *Client) ListChildren(ctx context.Context, ref string, limit int) ([]*Destination, error) {
	_, _, _, ok := parseDestRef(ref)
	if !ok {
		return nil, ErrUsage
	}
	selfKind, selfCC, selfSlug, _ := parseDestRef(ref)
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
			ParentRef:     selfID,
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
	var out []*Property
	seen := map[string]bool{}
	for _, a := range anchorRE.FindAllSubmatch(body, -1) {
		r := Classify(string(a[1]))
		if r.Kind != "property" || seen[r.ID] {
			continue
		}
		seen[r.ID] = true
		p := &Property{
			ID:         r.ID,
			Name:       anchorText(a[2]),
			URL:        URLFor("property", r.ID),
			ReviewsRef: r.ID,
		}
		if kind == "city" || kind == "district" {
			p.DestinationRef = selfID
		}
		out = append(out, p)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
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
