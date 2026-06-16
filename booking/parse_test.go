package booking

import "testing"

// These tests cover the pure parsers on small fixtures, so the mapping from HTML
// and JSON onto records is checked without a network round trip.

func TestClassifyAndURLForRoundTrip(t *testing.T) {
	ids := []struct{ kind, id string }{
		{"property", "gb/the-savoy"},
		{"destination", "country/us"},
		{"destination", "region/us/florida"},
		{"destination", "city/us/orlando"},
		{"destination", "district/gb/soho-london"},
	}
	for _, tc := range ids {
		u := URLFor(tc.kind, tc.id)
		if u == "" {
			t.Fatalf("URLFor(%q,%q) returned empty", tc.kind, tc.id)
		}
		r := Classify(u)
		if r.Kind != tc.kind || r.ID != tc.id {
			t.Errorf("Classify(URLFor(%q,%q)) = (%q,%q), want (%q,%q)",
				tc.kind, tc.id, r.Kind, r.ID, tc.kind, tc.id)
		}
	}
}

const propertyIslandHTML = `<!doctype html><html><head>
<script type="application/ld+json">
{"@type":"Hotel","name":"The Savoy","description":"A hotel on the Strand.",
 "image":["https://img/1.jpg","https://img/2.jpg"],
 "address":{"streetAddress":"Strand","addressLocality":"London","addressRegion":"England","postalCode":"WC2R 0EZ","addressCountry":"GB"},
 "geo":{"latitude":"51.51","longitude":"-0.12"},
 "aggregateRating":{"ratingValue":"9.1","reviewCount":"1200"},
 "starRating":{"ratingValue":"5"},
 "review":[{"author":"Sam","reviewBody":"Lovely stay.","datePublished":"2026-01-02","reviewRating":{"ratingValue":"10"}},
           {"author":{"name":"Jo"},"reviewBody":"Great location.","datePublished":"2026-02-03","reviewRating":{"ratingValue":"9"}}]}
</script>
</head><body>
<a href="/city/gb/london.html">London</a>
</body></html>`

func TestParsePropertyIsland(t *testing.T) {
	p := parsePropertyIsland([]byte(propertyIslandHTML), "gb/the-savoy", BaseURL+"/hotel/gb/the-savoy.html")
	if p == nil {
		t.Fatal("parsePropertyIsland returned nil")
	}
	if p.Name != "The Savoy" {
		t.Errorf("Name = %q", p.Name)
	}
	if p.Type != "hotel" {
		t.Errorf("Type = %q, want hotel", p.Type)
	}
	if p.Stars != 5 {
		t.Errorf("Stars = %d, want 5", p.Stars)
	}
	if p.Rating != 9.1 {
		t.Errorf("Rating = %v, want 9.1", p.Rating)
	}
	if p.ReviewCount != 1200 {
		t.Errorf("ReviewCount = %d, want 1200", p.ReviewCount)
	}
	if p.City != "London" || p.Country != "GB" {
		t.Errorf("address = %q, %q", p.City, p.Country)
	}
	if p.Lat == 0 || p.Lng == 0 {
		t.Errorf("geo = %v, %v", p.Lat, p.Lng)
	}
	if p.Image != "https://img/1.jpg" || len(p.Photos) != 2 {
		t.Errorf("images = %q, %v", p.Image, p.Photos)
	}
	if p.ReviewsRef != "gb/the-savoy" {
		t.Errorf("ReviewsRef = %q", p.ReviewsRef)
	}
	if p.DestinationRef != "city/gb/london" {
		t.Errorf("DestinationRef = %q, want city/gb/london", p.DestinationRef)
	}
}

func TestHotelDocReviews(t *testing.T) {
	doc := hotelDoc([]byte(propertyIslandHTML))
	if doc == nil {
		t.Fatal("hotelDoc returned nil")
	}
	if len(doc.Review) != 2 {
		t.Fatalf("got %d reviews, want 2", len(doc.Review))
	}
	if doc.Review[0].Author.Name != "Sam" {
		t.Errorf("review 0 author = %q, want Sam", doc.Review[0].Author.Name)
	}
	if doc.Review[1].Author.Name != "Jo" {
		t.Errorf("review 1 author = %q, want Jo (from object form)", doc.Review[1].Author.Name)
	}
}

const cityLandingHTML = `<!doctype html><html><head><title>Hotels in Orlando | Booking.com</title></head>
<body>
<h1>Orlando: 1,234 properties found</h1>
<a href="/district/us/downtown-orlando-us.html">Downtown Orlando</a>
<a href="/hotel/us/rosen-plaza.html">Rosen Plaza</a>
<a href="/hotel/us/grand-bohemian.html">Grand Bohemian</a>
<a href="/region/us/florida.html">Florida</a>
</body></html>`

func TestDestNameAndCount(t *testing.T) {
	if got := destName([]byte(cityLandingHTML), "orlando", "us"); got != "Orlando: 1,234 properties found" {
		t.Errorf("destName = %q", got)
	}
	if got := headlineCount([]byte(cityLandingHTML)); got != 1234 {
		t.Errorf("headlineCount = %d, want 1234", got)
	}
}

func TestNameFromSlug(t *testing.T) {
	if got := nameFromSlug("soho-london"); got != "Soho London" {
		t.Errorf("nameFromSlug = %q, want Soho London", got)
	}
}

func TestMapSuggestion(t *testing.T) {
	m := autocompleteMatch{
		Label:    "Orlando",
		Type:     "city",
		Country:  "United States",
		NrHotels: 1234,
		URL:      "/city/us/orlando.html",
	}
	s := mapSuggestion("orl", m)
	if s == nil {
		t.Fatal("mapSuggestion returned nil")
	}
	if s.Destination != "city/us/orlando" {
		t.Errorf("Destination = %q, want city/us/orlando", s.Destination)
	}
	if s.SearchRef != "Orlando" {
		t.Errorf("SearchRef = %q, want Orlando", s.SearchRef)
	}
	if s.Property != "" {
		t.Errorf("Property = %q, want empty for a place", s.Property)
	}

	hotel := autocompleteMatch{Label: "The Savoy", Type: "hotel", CC: "gb", HotelSlug: "the-savoy"}
	hs := mapSuggestion("sav", hotel)
	if hs.Property != "gb/the-savoy" {
		t.Errorf("hotel Property = %q, want gb/the-savoy", hs.Property)
	}
}
