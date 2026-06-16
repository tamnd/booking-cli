package booking

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
)

// island.go reads one property from its page's JSON-LD island. The island is a
// schema.org Hotel or LodgingBusiness object and is the authoritative anonymous
// source for a property's static detail. The property page is the best-effort
// tier: a datacenter request may be met by the bot manager, in which case get
// returns ErrBlocked before the parser runs.

// hotelTypes are the schema.org @type values that mark the property island.
var hotelTypes = map[string]bool{
	"hotel": true, "lodgingbusiness": true, "resort": true,
	"bedandbreakfast": true, "hostel": true, "motel": true,
	"apartment": true, "vacationrental": true,
}

// cityLinkRE finds the first /city/<cc>/<slug> landing link in a page body, used
// to set a property's DestinationRef from a real link rather than a guessed slug.
var cityLinkRE = regexp.MustCompile(`/city/([a-z]{2})/([^"./?]+)\.html`)

// GetProperty fetches a property by id ("<cc>/<slug>") or URL and returns it.
func (c *Client) GetProperty(ctx context.Context, ref string) (*Property, error) {
	id := propertyID(ref)
	if id == "" {
		return nil, ErrUsage
	}
	pageURL := URLFor("property", id)
	body, err := c.get(ctx, pageURL)
	if err != nil {
		return nil, err
	}
	p := parsePropertyIsland(body, id, pageURL)
	if p == nil {
		return nil, ErrNotFound
	}
	return p, nil
}

// propertyID normalizes a property reference to its "<cc>/<slug>" id.
func propertyID(ref string) string {
	r := Classify(ref)
	if r.Kind == "property" {
		return r.ID
	}
	return ""
}

// parsePropertyIsland maps the schema.org island onto a Property. It returns nil
// when the body has no recognizable hotel island.
func parsePropertyIsland(body []byte, id, pageURL string) *Property {
	doc := hotelDoc(body)
	if doc == nil {
		return nil
	}
	p := &Property{
		ID:          id,
		Name:        squish(doc.Name),
		Description: squish(doc.Description),
		Street:      squish(doc.Address.StreetAddress),
		City:        squish(doc.Address.AddressLocality),
		Region:      squish(doc.Address.AddressRegion),
		Zip:         squish(doc.Address.PostalCode),
		Country:     squish(doc.Address.AddressCountry),
		Lat:         doc.Geo.Latitude.float(),
		Lng:         doc.Geo.Longitude.float(),
		Rating:      doc.AggregateRating.RatingValue.float(),
		ReviewCount: doc.AggregateRating.ReviewCount.int(),
		Stars:       doc.StarRating.RatingValue.int(),
		Type:        accommodationType(string(doc.Type)),
		Photos:      doc.Image,
		PriceRange:  squish(doc.PriceRange),
		Phone:       squish(doc.Telephone),
		Map:         doc.HasMap.string(),
		CheckIn:     squish(doc.CheckinTime),
		CheckOut:    squish(doc.CheckoutTime),
		Amenities:   amenityNames(doc.AmenityFeature),
		URL:         pageURL,
		ReviewsRef:  id,
	}
	if len(doc.Image) > 0 {
		p.Image = doc.Image[0]
	}
	if a := doc.Address; a.StreetAddress != "" || a.AddressLocality != "" {
		p.DisplayAddress = compact([]string{a.StreetAddress, a.AddressLocality, a.AddressRegion, a.PostalCode, a.AddressCountry})
	}
	// DestinationRef from a real city landing link on the page, not a guessed slug.
	if m := cityLinkRE.FindSubmatch(body); m != nil {
		p.DestinationRef = "city/" + string(m[1]) + "/" + string(m[2])
	}
	return p
}

// hotelDoc returns the first ld+json block whose @type is a hotel-like type.
func hotelDoc(body []byte) *ldDoc {
	for _, raw := range ldBlocks(body) {
		var doc ldDoc
		if err := json.Unmarshal(raw, &doc); err != nil {
			continue
		}
		if hotelTypes[strings.ToLower(string(doc.Type))] {
			return &doc
		}
	}
	return nil
}

// accommodationType maps a schema.org @type to Booking's plain word.
func accommodationType(t string) string {
	switch strings.ToLower(t) {
	case "hotel", "":
		if t == "" {
			return ""
		}
		return "hotel"
	case "lodgingbusiness":
		return "hotel"
	case "bedandbreakfast":
		return "bnb"
	case "vacationrental":
		return "villa"
	default:
		return strings.ToLower(t)
	}
}

// amenityNames returns the names of the amenities the island marks as available,
// deduped and in island order, dropping any with a falsey value.
func amenityNames(feats []ldFeature) []string {
	var out []string
	seen := map[string]bool{}
	for _, f := range feats {
		name := squish(f.Name)
		if name == "" || !f.Value.truthy() || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

// compact drops empty strings from a slice.
func compact(in []string) []string {
	var out []string
	for _, s := range in {
		if s = squish(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}
