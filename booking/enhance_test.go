package booking

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// enhance_test.go covers the deep-audit additions: the sitemap crawl root, the
// landing-page geo read, the property-island amenities and side fields, the card
// image merge, and the 202 bot wall.

func TestSitemapRoundTrip(t *testing.T) {
	// Classify recognizes a per-kind index URL, and URLFor rebuilds it.
	u := URLFor("sitemap", "country")
	if u != BaseURL+"/sitembk-country-index.xml" {
		t.Fatalf("URLFor sitemap = %q", u)
	}
	r := Classify(u)
	if r.Kind != "sitemap" || r.ID != "country" {
		t.Errorf("Classify(%q) = (%q,%q), want (sitemap,country)", u, r.Kind, r.ID)
	}
}

func TestSeedFor(t *testing.T) {
	place := seedFor("city", BaseURL+"/city/us/orlando.html", "2026-01-01")
	if place == nil || place.Destination != "city/us/orlando" || place.Property != "" {
		t.Errorf("place seed = %+v", place)
	}
	if place.Lastmod != "2026-01-01" {
		t.Errorf("lastmod = %q", place.Lastmod)
	}
	hotel := seedFor("hotel", BaseURL+"/hotel/gb/the-savoy.html", "")
	if hotel == nil || hotel.Property != "gb/the-savoy" || hotel.Destination != "" {
		t.Errorf("hotel seed = %+v", hotel)
	}
	// A hotel-review page resolves to the property it reviews.
	rev := seedFor("hotel-review", BaseURL+"/reviews/gb/hotel/the-savoy.en-gb.html", "")
	if rev == nil || rev.Property != "gb/the-savoy" || rev.Destination != "" {
		t.Errorf("review seed = %+v", rev)
	}
	// A page outside the accommodations graph still comes back, carrying the URL
	// and lastmod with no edge, so the inventory stays complete.
	other := seedFor("attractions", BaseURL+"/attractions/us/disney.html", "2026-02-02")
	if other == nil {
		t.Fatal("unrecognized URL should still yield a seed")
	}
	if other.Destination != "" || other.Property != "" {
		t.Errorf("unrecognized seed should carry no edge, got %+v", other)
	}
	if other.URL != BaseURL+"/attractions/us/disney.html" || other.Lastmod != "2026-02-02" {
		t.Errorf("unrecognized seed should keep url+lastmod, got %+v", other)
	}
}

func TestReviewsPageClassify(t *testing.T) {
	// The dedicated /reviews/<cc>/hotel/<slug> page resolves to its property.
	r := Classify(BaseURL + "/reviews/gb/hotel/the-savoy.en-gb.html")
	if r.Kind != "property" || r.ID != "gb/the-savoy" {
		t.Errorf("Classify reviews page = (%q,%q), want (property,gb/the-savoy)", r.Kind, r.ID)
	}
}

func TestSitemapsDiscovery(t *testing.T) {
	robots := strings.Join([]string{
		"User-agent: *",
		"Disallow: /searchresults",
		"Sitemap: https://www.booking.com/sitembk-country-index.xml",
		"Sitemap: https://www.booking.com/sitembk-hotel-index.xml",
		"Sitemap: https://www.booking.com/sitembk-hotel-review-index.xml",
		"Sitemap: https://www.booking.com/sitembk-themed-city-ski-index.xml",
		"Sitemap: https://www.booking.com/sitembk-attractions-index.xml",
		"Sitemap: https://www.booking.com/sitemap.xml",
		// A duplicate is dropped.
		"Sitemap: https://www.booking.com/sitembk-country-index.xml",
	}, "\n")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(robots))
	}))
	defer srv.Close()

	c := NewClient(Config{NoCache: true})
	c.HTTP = srv.Client()
	idxs, err := c.sitemapsFrom(context.Background(), srv.URL, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(idxs) != 6 {
		t.Fatalf("got %d indexes, want 6", len(idxs))
	}
	want := map[string]struct{ kind, category string }{
		"https://www.booking.com/sitembk-country-index.xml":         {"country", "place"},
		"https://www.booking.com/sitembk-hotel-index.xml":           {"hotel", "property"},
		"https://www.booking.com/sitembk-hotel-review-index.xml":    {"hotel-review", "reviews"},
		"https://www.booking.com/sitembk-themed-city-ski-index.xml": {"themed-city-ski", "theme"},
		"https://www.booking.com/sitembk-attractions-index.xml":     {"attractions", "attraction"},
		"https://www.booking.com/sitemap.xml":                       {"", ""},
	}
	for _, idx := range idxs {
		w, ok := want[idx.URL]
		if !ok {
			t.Errorf("unexpected index %q", idx.URL)
			continue
		}
		if idx.Kind != w.kind || idx.Category != w.category {
			t.Errorf("index %q = (%q,%q), want (%q,%q)", idx.URL, idx.Kind, idx.Category, w.kind, w.category)
		}
		if w.kind != "" && idx.SeedsRef != w.kind {
			t.Errorf("index %q SeedsRef = %q, want %q", idx.URL, idx.SeedsRef, w.kind)
		}
	}
}

func TestIndexKind(t *testing.T) {
	cases := map[string]string{
		"https://www.booking.com/sitembk-country-index.xml":         "country",
		"https://www.booking.com/sitembk-themed-city-ski-index.xml": "themed-city-ski",
		"sitembk-hotel-review-index.xml":                            "hotel-review",
		"https://www.booking.com/sitemap.xml":                       "",
		"https://www.booking.com/sitembk--index.xml":                "",
	}
	for in, want := range cases {
		if got := indexKind(in); got != want {
			t.Errorf("indexKind(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSitemapCategory(t *testing.T) {
	cases := map[string]string{
		"country":           "place",
		"regiongroup":       "place",
		"airport":           "place",
		"hotel":             "property",
		"hotel-review":      "reviews",
		"attractions":       "attraction",
		"beaches":           "beach",
		"themed-city-ski":   "theme",
		"cars":              "car",
		"flights":           "flight",
		"articles":          "article",
		"something-unknown": "other",
	}
	for kind, want := range cases {
		if got := sitemapCategory(kind); got != want {
			t.Errorf("sitemapCategory(%q) = %q, want %q", kind, got, want)
		}
	}
}

func TestRobotsClassify(t *testing.T) {
	// robots.txt is the master list, classified as sitemaps with no id.
	r := Classify(BaseURL + "/robots.txt")
	if r.Kind != "sitemaps" || r.ID != "" {
		t.Errorf("Classify robots = (%q,%q), want (sitemaps,)", r.Kind, r.ID)
	}
	if URLFor("sitemaps", "") != BaseURL+"/robots.txt" {
		t.Errorf("URLFor sitemaps = %q", URLFor("sitemaps", ""))
	}
}

func TestSelectShards(t *testing.T) {
	idx := xmlIndex{}
	idx.Sitemaps = []struct {
		Loc string `xml:"loc"`
	}{
		{BaseURL + "/sitembk-country-en-us.0000.xml.gz"},
		{BaseURL + "/sitembk-country-fr-fr.0000.xml.gz"},
		{BaseURL + "/sitembk-country-en-us.0001.xml.gz"},
	}
	got := selectShards(idx, "en-us")
	if len(got) != 2 {
		t.Fatalf("en-us shards = %v, want 2", got)
	}
	// An unmatched locale falls back to all shards rather than none.
	if all := selectShards(idx, "zz-zz"); len(all) != 3 {
		t.Errorf("fallback shards = %v, want 3", all)
	}
}

func TestMaybeGunzip(t *testing.T) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte("<urlset></urlset>"))
	_ = zw.Close()
	if got := maybeGunzip(buf.Bytes()); string(got) != "<urlset></urlset>" {
		t.Errorf("gunzip = %q", got)
	}
	plain := []byte("<sitemapindex></sitemapindex>")
	if got := maybeGunzip(plain); string(got) != string(plain) {
		t.Errorf("plain passthrough = %q", got)
	}
}

func TestSitemapEndToEnd(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/sitembk-city-index.xml", func(w http.ResponseWriter, r *http.Request) {
		shard := "http://" + r.Host + "/shard.xml.gz"
		_, _ = w.Write([]byte(`<sitemapindex><sitemap><loc>` + shard + `</loc></sitemap></sitemapindex>`))
	})
	mux.HandleFunc("/shard.xml.gz", func(w http.ResponseWriter, _ *http.Request) {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, _ = zw.Write([]byte(`<urlset>
			<url><loc>https://www.booking.com/city/us/orlando.html</loc><lastmod>2026-01-01</lastmod></url>
			<url><loc>https://www.booking.com/city/us/miami.html</loc></url>
		</urlset>`))
		_ = zw.Close()
		_, _ = w.Write(buf.Bytes())
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(Config{NoCache: true, Locale: "en-us"})
	// Point the index at the test server by overriding the host through the URL the
	// client builds: Sitemap reads BaseURL, so exercise it via the index handler by
	// fetching through a client whose requests we redirect with a custom transport.
	c.HTTP = srv.Client()
	seeds, err := c.sitemapFrom(context.Background(), srv.URL, "city", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(seeds) != 2 {
		t.Fatalf("got %d seeds, want 2", len(seeds))
	}
	if seeds[0].Destination != "city/us/orlando" || seeds[0].Lastmod != "2026-01-01" {
		t.Errorf("seed 0 = %+v", seeds[0])
	}
}

func TestLandingGeo(t *testing.T) {
	html := `<html><head>
	<script type="application/ld+json">{"@type":"WebPage","name":"x"}</script>
	<script type="application/ld+json">{"@type":"City","@id":"https://www.booking.com/#/schema/City/miami","name":"Miami","geo":{"latitude":25.77,"longitude":-80.19}}</script>
	</head></html>`
	lat, lng := landingGeo([]byte(html), "miami")
	if lat != 25.77 || lng != -80.19 {
		t.Errorf("geo = %v,%v want 25.77,-80.19", lat, lng)
	}
	if lat, lng := landingGeo([]byte(`<html></html>`), "x"); lat != 0 || lng != 0 {
		t.Errorf("empty page geo = %v,%v want 0,0", lat, lng)
	}
}

const amenityIslandHTML = `<html><head>
<script type="application/ld+json">
{"@type":"Hotel","name":"Test Inn","priceRange":"$$","telephone":"+1 305 555 0100",
 "hasMap":{"@type":"Map","url":"https://maps.example/x"},
 "checkinTime":"15:00","checkoutTime":"11:00",
 "amenityFeature":[
   {"name":"Free WiFi","value":true},
   {"name":"Pool","value":"1"},
   {"name":"Spa","value":false},
   {"name":"Parking"}
 ]}
</script></head></html>`

func TestParseIslandSideFields(t *testing.T) {
	p := parsePropertyIsland([]byte(amenityIslandHTML), "us/test-inn", BaseURL+"/hotel/us/test-inn.html")
	if p == nil {
		t.Fatal("nil property")
	}
	if p.PriceRange != "$$" {
		t.Errorf("PriceRange = %q", p.PriceRange)
	}
	if p.Phone != "+1 305 555 0100" {
		t.Errorf("Phone = %q", p.Phone)
	}
	if p.Map != "https://maps.example/x" {
		t.Errorf("Map = %q", p.Map)
	}
	if p.CheckIn != "15:00" || p.CheckOut != "11:00" {
		t.Errorf("checkin/out = %q/%q", p.CheckIn, p.CheckOut)
	}
	// Free WiFi (true), Pool ("1"), and Parking (no value -> present) are kept; Spa
	// (false) is dropped.
	want := []string{"Free WiFi", "Pool", "Parking"}
	if strings.Join(p.Amenities, "|") != strings.Join(want, "|") {
		t.Errorf("Amenities = %v, want %v", p.Amenities, want)
	}
}

func TestPropertyCardsMergeImage(t *testing.T) {
	html := `<html><body>
	<a href="/hotel/us/rosen-plaza.html"><img data-src="https://cf.bstatic.com/xdata/images/hotel/rosen.jpg"></a>
	<a href="/hotel/us/rosen-plaza.html">Rosen Plaza</a>
	<a href="/hotel/us/grand.html">Grand Bohemian</a>
	</body></html>`
	cards := propertyCards([]byte(html))
	if len(cards) != 2 {
		t.Fatalf("got %d cards, want 2", len(cards))
	}
	if cards[0].ID != "us/rosen-plaza" || cards[0].Name != "Rosen Plaza" {
		t.Errorf("card 0 = %+v", cards[0])
	}
	if cards[0].Image != "https://cf.bstatic.com/xdata/images/hotel/rosen.jpg" {
		t.Errorf("card 0 image = %q", cards[0].Image)
	}
}

func TestGetBlockedOn202(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()
	if _, err := testClient().get(context.Background(), srv.URL); err != ErrBlocked {
		t.Errorf("err = %v, want ErrBlocked for 202", err)
	}
}
