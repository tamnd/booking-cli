package booking

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strconv"
)

// jsonld.go extracts and decodes the schema.org JSON-LD islands a Booking.com
// property page embeds, with flexible field types because the island plays loose
// with JSON: a field that is sometimes a string is sometimes an array, and a
// number is sometimes quoted. The helpers below absorb that so the parsers in
// island.go and reviews.go map clean values.

var ldScriptRE = regexp.MustCompile(`(?is)<script[^>]+type="application/ld\+json"[^>]*>(.*?)</script>`)

// ldBlocks returns the raw JSON of every ld+json script in an HTML body.
func ldBlocks(body []byte) [][]byte {
	var out [][]byte
	for _, m := range ldScriptRE.FindAllSubmatch(body, -1) {
		out = append(out, bytes.TrimSpace(m[1]))
	}
	return out
}

// ldDoc is the subset of a schema.org Hotel/LodgingBusiness island booking-cli
// reads.
type ldDoc struct {
	Type            jsonType   `json:"@type"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	URL             string     `json:"url"`
	Image           jsonStr    `json:"image"`
	PriceRange      string     `json:"priceRange"`
	Telephone       string     `json:"telephone"`
	Address         ldAddress  `json:"address"`
	Geo             ldGeo      `json:"geo"`
	AggregateRating ldAgg      `json:"aggregateRating"`
	StarRating      ldRating   `json:"starRating"`
	Review          []ldReview `json:"review"`
}

type ldAddress struct {
	StreetAddress   string `json:"streetAddress"`
	AddressLocality string `json:"addressLocality"`
	AddressRegion   string `json:"addressRegion"`
	PostalCode      string `json:"postalCode"`
	AddressCountry  string `json:"addressCountry"`
}

type ldGeo struct {
	Latitude  fnum `json:"latitude"`
	Longitude fnum `json:"longitude"`
}

type ldAgg struct {
	RatingValue fnum `json:"ratingValue"`
	ReviewCount fnum `json:"reviewCount"`
	BestRating  fnum `json:"bestRating"`
}

type ldRating struct {
	RatingValue fnum `json:"ratingValue"`
}

type ldReview struct {
	Author        ldAuthor `json:"author"`
	ReviewBody    string   `json:"reviewBody"`
	Name          string   `json:"name"`
	DatePublished string   `json:"datePublished"`
	InLanguage    string   `json:"inLanguage"`
	ReviewRating  ldRating `json:"reviewRating"`
}

// ldAuthor decodes an author that is either a bare string or a {name} object.
type ldAuthor struct {
	Name string
}

func (a *ldAuthor) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		return nil
	}
	if b[0] == '"' {
		return json.Unmarshal(b, &a.Name)
	}
	var obj struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return nil // tolerate odd shapes
	}
	a.Name = obj.Name
	return nil
}

// jsonType decodes @type whether it is a string or an array of strings, keeping
// the first.
type jsonType string

func (t *jsonType) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		return nil
	}
	if b[0] == '[' {
		var arr []string
		if err := json.Unmarshal(b, &arr); err != nil {
			return nil
		}
		if len(arr) > 0 {
			*t = jsonType(arr[0])
		}
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return nil
	}
	*t = jsonType(s)
	return nil
}

// jsonStr decodes a field that is either a string or an array of strings into a
// slice.
type jsonStr []string

func (j *jsonStr) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		return nil
	}
	if b[0] == '[' {
		var arr []string
		if err := json.Unmarshal(b, &arr); err != nil {
			return nil
		}
		*j = arr
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return nil
	}
	if s != "" {
		*j = []string{s}
	}
	return nil
}

// fnum decodes a number that may be quoted, e.g. 9.2 or "9.2".
type fnum float64

func (f *fnum) UnmarshalJSON(b []byte) error {
	b = bytes.Trim(bytes.TrimSpace(b), `"`)
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		return nil
	}
	v, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		return nil
	}
	*f = fnum(v)
	return nil
}

func (f fnum) float() float64 { return float64(f) }
func (f fnum) int() int       { return int(float64(f)) }
