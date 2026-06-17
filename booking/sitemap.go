package booking

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/xml"
	"io"
	"regexp"
	"strings"
)

// sitemap.go reads Booking.com's published sitemaps, the reconstruction backbone
// for a crawl. robots.txt lists a per-kind index at /sitembk-<kind>-index.xml;
// each index is a sitemapindex of per-language shards (e.g.
// sitembk-country-en-us.0000.xml.gz); each shard is a gzipped urlset enumerating
// every landing page of that kind. Two operations read this:
//
//   - sitemaps walks robots.txt and lists every published index, so a crawl can
//     discover the whole backbone with no prior knowledge of what kinds exist.
//   - sitemap reads one index, picks the shards for the locale, gunzips and parses
//     them, and emits a Seed per page.
//
// Together they are the crawl root. A Seed needs no prior id, so a crawl starts at
// sitemaps, follows each index into its seeds, and then follows the record edges
// (a place seed into its destination, a hotel seed into its property) to reach the
// rest of the public site. Pages outside the accommodations graph (attractions,
// beaches, themed lists) still come back as seeds carrying the URL and lastmod, so
// no page is dropped from the reconstruction even when it maps to no typed record.
//
// The sitemaps live on the reliable estate host and read from anywhere; only the
// property and review detail behind them is best-effort.

// kindRE guards the kind segment of an index name against anything but the lower
// snake/kebab tokens Booking uses, so a kind cannot smuggle a path into the URL.
var kindRE = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// robotsSitemapRE pulls each advertised index URL out of robots.txt.
var robotsSitemapRE = regexp.MustCompile(`(?mi)^\s*Sitemap:\s*(\S+)\s*$`)

// indexNameRE matches the canonical per-kind index filename and captures the kind,
// e.g. "country" in "sitembk-country-index.xml" or "themed-city-ski" in
// "sitembk-themed-city-ski-index.xml".
var indexNameRE = regexp.MustCompile(`^sitembk-(.+)-index\.xml$`)

// xmlIndex is a sitemapindex: a list of shard URLs.
type xmlIndex struct {
	Sitemaps []struct {
		Loc string `xml:"loc"`
	} `xml:"sitemap"`
}

// xmlURLSet is a urlset: the entries of one shard.
type xmlURLSet struct {
	URLs []struct {
		Loc     string `xml:"loc"`
		Lastmod string `xml:"lastmod"`
	} `xml:"url"`
}

// Sitemaps lists the sitemap indexes Booking advertises in robots.txt, the
// root-of-roots: each record names one index, the kind it enumerates, and the
// category of page it covers. SeedsRef links into the sitemap op for that kind, so
// a crawl walks from here into every backbone the site publishes.
func (c *Client) Sitemaps(ctx context.Context, limit int) ([]*SitemapIndex, error) {
	return c.sitemapsFrom(ctx, BaseURL, limit)
}

// sitemapsFrom is Sitemaps with the host parameterized for tests.
func (c *Client) sitemapsFrom(ctx context.Context, base string, limit int) ([]*SitemapIndex, error) {
	body, err := c.get(ctx, base+"/robots.txt")
	if err != nil {
		return nil, err
	}
	var out []*SitemapIndex
	seen := map[string]bool{}
	for _, m := range robotsSitemapRE.FindAllSubmatch(body, -1) {
		loc := strings.TrimSpace(string(m[1]))
		if loc == "" || seen[loc] {
			continue
		}
		seen[loc] = true
		idx := &SitemapIndex{URL: loc}
		if kind := indexKind(loc); kind != "" {
			idx.Kind = kind
			idx.Category = sitemapCategory(kind)
			idx.SeedsRef = kind
		}
		out = append(out, idx)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

// indexKind returns the kind a /sitembk-<kind>-index.xml URL enumerates, or "" for
// an index whose name does not fit that template (the few language master indexes
// the sitemap op cannot rebuild a URL for).
func indexKind(loc string) string {
	name := loc
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	m := indexNameRE.FindStringSubmatch(name)
	if m == nil || !kindRE.MatchString(m[1]) {
		return ""
	}
	return m[1]
}

// sitemapCategory groups a kind into the kind of page it covers, so a crawl can
// pick the backbones it cares about. The accommodations graph is place, property,
// and reviews; the rest are pages Booking publishes alongside it.
func sitemapCategory(kind string) string {
	switch {
	case kind == "country", kind == "region", kind == "regiongroup",
		kind == "city", kind == "district", kind == "landmark", kind == "airport":
		return "place"
	case kind == "hotel":
		return "property"
	case kind == "hotel-review" || strings.HasPrefix(kind, "reviews"):
		return "reviews"
	case strings.HasPrefix(kind, "attractions"):
		return "attraction"
	case kind == "beaches" || kind == "beach-holidays":
		return "beach"
	case strings.HasPrefix(kind, "themed"):
		return "theme"
	case strings.HasPrefix(kind, "cars"):
		return "car"
	case strings.HasPrefix(kind, "flights"):
		return "flight"
	case kind == "articles", kind == "editorial-articles", kind == "product-guides",
		kind == "tourism", kind == "newly-opened", kind == "holidays-city",
		strings.HasPrefix(kind, "extended-stays"), strings.HasPrefix(kind, "discover"):
		return "article"
	default:
		return "other"
	}
}

// Sitemap returns up to limit seeds for a kind by reading its sitemap index and
// the locale's shards. It is the crawl root: every seed names a live landing page
// and, when the page maps to a record, the edge into it.
func (c *Client) Sitemap(ctx context.Context, kind string, limit int) ([]*Seed, error) {
	return c.sitemapFrom(ctx, BaseURL, kind, limit)
}

// sitemapFrom is Sitemap with the index host parameterized, so a test can point
// the index at a local server while the shard URLs in the index stay absolute.
func (c *Client) sitemapFrom(ctx context.Context, base, kind string, limit int) ([]*Seed, error) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if !kindRE.MatchString(kind) {
		return nil, ErrUsage
	}
	body, err := c.get(ctx, base+"/sitembk-"+kind+"-index.xml")
	if err != nil {
		return nil, err
	}
	var idx xmlIndex
	if err := xml.Unmarshal(maybeGunzip(body), &idx); err != nil {
		return nil, ErrNotFound
	}
	shards := selectShards(idx, c.cfg.Locale)
	if len(shards) == 0 {
		return nil, nil
	}

	var out []*Seed
	seen := map[string]bool{}
	for _, shardURL := range shards {
		sb, err := c.get(ctx, shardURL)
		if err != nil {
			// One bad shard should not sink the whole walk; move to the next.
			continue
		}
		var set xmlURLSet
		if err := xml.Unmarshal(maybeGunzip(sb), &set); err != nil {
			continue
		}
		for _, u := range set.URLs {
			loc := strings.TrimSpace(u.Loc)
			if loc == "" || seen[loc] {
				continue
			}
			seen[loc] = true
			out = append(out, seedFor(kind, loc, strings.TrimSpace(u.Lastmod)))
			if limit > 0 && len(out) >= limit {
				return out, nil
			}
		}
	}
	return out, nil
}

// selectShards returns the shard URLs to read for a locale, preferring those
// whose name carries the locale token (e.g. "-en-us.") and falling back to every
// shard when none match, so an odd locale still reconstructs the estate.
func selectShards(idx xmlIndex, locale string) []string {
	locale = strings.ToLower(strings.TrimSpace(locale))
	var matched, all []string
	for _, s := range idx.Sitemaps {
		loc := strings.TrimSpace(s.Loc)
		if loc == "" {
			continue
		}
		all = append(all, loc)
		if locale != "" && strings.Contains(strings.ToLower(loc), "-"+locale+".") {
			matched = append(matched, loc)
		}
	}
	if len(matched) > 0 {
		return matched
	}
	return all
}

// seedFor builds a Seed from a sitemap entry. It always emits the page, since the
// point of the backbone is a complete URL inventory, and wires the edge into the
// graph when the reference layer recognizes the URL: a place page fills
// Destination, a property or its reviews page fills Property. A page outside the
// accommodations graph (an attraction, beach, or themed list) comes back with the
// URL and lastmod and no edge.
func seedFor(kind, loc, lastmod string) *Seed {
	r := Classify(loc)
	s := &Seed{Kind: kind, ID: r.ID, URL: loc, Lastmod: lastmod}
	switch r.Kind {
	case "destination":
		s.Destination = r.ID
	case "property":
		s.Property = r.ID
	}
	return s
}

// maybeGunzip returns the gzip-decompressed body when it carries the gzip magic
// bytes, else the body unchanged, so a .gz shard and a plain .xml index both
// parse through one path.
func maybeGunzip(b []byte) []byte {
	if len(b) < 2 || b[0] != 0x1f || b[1] != 0x8b {
		return b
	}
	zr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return b
	}
	defer func() { _ = zr.Close() }()
	out, err := io.ReadAll(zr)
	if err != nil {
		return b
	}
	return out
}
