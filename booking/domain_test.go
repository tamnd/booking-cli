package booking

import (
	"testing"

	"github.com/tamnd/any-cli/kit"
)

// These tests are offline: they exercise the URI driver's pure string functions
// and the host wiring (mint, body, resolve), which need no network. The client's
// HTTP behaviour is covered in booking_test.go.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "booking" {
		t.Errorf("Scheme = %q, want booking", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "booking" {
		t.Errorf("Identity.Binary = %q, want booking", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct{ in, typ, id string }{
		{"gb/the-savoy", "property", "gb/the-savoy"},
		{"/hotel/gb/the-savoy.html", "property", "gb/the-savoy"},
		{"https://" + Host + "/hotel/us/plaza.en-gb.html", "property", "us/plaza"},
		{"https://" + Host + "/country/us.html", "destination", "country/us"},
		{"/region/us/florida.html", "destination", "region/us/florida"},
		{"https://" + Host + "/city/us/orlando.html", "destination", "city/us/orlando"},
		{"https://" + Host + "/searchresults.html?ss=Orlando", "search", "Orlando"},
	}
	for _, tc := range cases {
		typ, id, err := Domain{}.Classify(tc.in)
		if err != nil || typ != tc.typ || id != tc.id {
			t.Errorf("Classify(%q) = (%q, %q, %v), want (%q, %q, nil)",
				tc.in, typ, id, err, tc.typ, tc.id)
		}
	}
}

func TestClassifyUnknown(t *testing.T) {
	if _, _, err := (Domain{}).Classify("not a booking reference at all"); err == nil {
		t.Error("Classify of an unknown reference returned no error")
	}
}

func TestLocate(t *testing.T) {
	cases := []struct{ typ, id, want string }{
		{"property", "gb/the-savoy", BaseURL + "/hotel/gb/the-savoy.html"},
		{"reviews", "gb/the-savoy", BaseURL + "/hotel/gb/the-savoy.html"},
		{"destination", "country/us", BaseURL + "/country/us.html"},
		{"destination", "city/us/orlando", BaseURL + "/city/us/orlando.html"},
		{"search", "Orlando", BaseURL + "/searchresults.html?ss=Orlando"},
	}
	for _, tc := range cases {
		got, err := Domain{}.Locate(tc.typ, tc.id)
		if err != nil || got != tc.want {
			t.Errorf("Locate(%q,%q) = (%q, %v), want (%q, nil)", tc.typ, tc.id, got, err, tc.want)
		}
	}
}

// TestHostWiring mounts the driver in a kit Host (the runtime ant drives) and
// checks the round trip: a record mints to its URI, its body is readable, and a
// bare id resolves back to the same URI. The init in domain.go registers the
// domain, so kit.Open finds it.
func TestHostWiring(t *testing.T) {
	h, err := kit.Open()
	if err != nil {
		t.Fatal(err)
	}

	p := &Property{
		ID:          "gb/the-savoy",
		Name:        "The Savoy",
		URL:         BaseURL + "/hotel/gb/the-savoy.html",
		Description: "A hotel on the Strand.",
	}
	u, err := h.Mint(p)
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	if want := "booking://property/gb/the-savoy"; u.String() != want {
		t.Errorf("Mint = %q, want %q", u.String(), want)
	}

	if body, ok := h.Body(p); !ok || body == "" {
		t.Errorf("Body = (%q, %v), want non-empty", body, ok)
	}

	if !h.Searchable("booking") {
		t.Error("Searchable = false, want true (the domain registers a search op)")
	}

	got, err := h.ResolveOn("booking", "gb/the-savoy")
	if err != nil || got.String() != "booking://property/gb/the-savoy" {
		t.Errorf("ResolveOn = (%q, %v), want booking://property/gb/the-savoy", got.String(), err)
	}
}
