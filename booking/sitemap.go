package booking

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/xml"
	"io"
	"strings"
)

// sitemap.go reads Booking.com's published sitemaps, the reconstruction backbone
// for a crawl. robots.txt advertises a per-kind index at
// /sitembk-<kind>-index.xml; that index is a sitemapindex listing per-language
// shards (e.g. sitembk-country-en-us.0000.xml.gz); each shard is a gzipped XML
// urlset enumerating every landing page of that kind. Sitemap walks the index,
// picks the shards for the client's locale, gunzips and parses them, and turns
// each entry into a Seed that carries the edge into the rest of the graph. A
// crawl that starts here needs no prior id: it fans from the seeds into every
// destination and property and then follows the record edges to reach the rest.
//
// The sitemaps live on the reliable estate host and read from anywhere; only the
// property and review detail behind them is best-effort.

// sitemapKinds are the kinds Booking publishes a sitemap index for. The six place
// kinds map onto destination nodes; hotel maps onto properties.
var sitemapKinds = map[string]bool{
	"country": true, "region": true, "city": true,
	"district": true, "landmark": true, "airport": true,
	"hotel": true,
}

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

// Sitemap returns up to limit seeds for a kind by reading its sitemap index and
// the locale's shards. It is the crawl root: every seed names a live landing page
// and the record it points at.
func (c *Client) Sitemap(ctx context.Context, kind string, limit int) ([]*Seed, error) {
	return c.sitemapFrom(ctx, BaseURL, kind, limit)
}

// sitemapFrom is Sitemap with the index host parameterized, so a test can point
// the index at a local server while the shard URLs in the index stay absolute.
func (c *Client) sitemapFrom(ctx context.Context, base, kind string, limit int) ([]*Seed, error) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if !sitemapKinds[kind] {
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
			s := seedFor(kind, loc, strings.TrimSpace(u.Lastmod))
			if s == nil {
				continue
			}
			out = append(out, s)
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

// seedFor builds a Seed from a sitemap entry, classifying the URL and wiring the
// edge into the graph: a place URL fills Destination, a hotel URL fills Property.
// It returns nil for a URL the reference layer does not recognize.
func seedFor(kind, loc, lastmod string) *Seed {
	r := Classify(loc)
	s := &Seed{Kind: kind, ID: r.ID, URL: loc, Lastmod: lastmod}
	switch r.Kind {
	case "destination":
		s.Destination = r.ID
	case "property":
		s.Property = r.ID
	default:
		return nil
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
