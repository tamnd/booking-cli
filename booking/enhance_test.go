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
	if bad := seedFor("city", "https://www.booking.com/some/other/path", ""); bad != nil {
		t.Errorf("unrecognized URL should yield nil, got %+v", bad)
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
