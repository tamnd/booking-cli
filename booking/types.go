package booking

// This file holds the exported records the commands emit. Their json tags name
// the fields a reader sees, kit:"id" marks the key the record store upserts on,
// kit:"body" marks the long-text field `booking cat` and the Markdown export
// print, and table:",truncate" keeps wide free text from blowing up a terminal
// table. Each record carries only fields a logged-out reader can fill: no owner
// dashboards, no partner rates, no signed-in surfaces, because none of that is
// reachable without an account. There is no Rank column; emit order is the rank.
//
// The kit:"link" edges connect the records into one graph a host walks for
// breadth-first crawls, and they are what lets a crawl reconstruct the reachable
// public site from a single seed. A resolver edge (property, destination) names a
// bare field and points at one record; a collection edge carries the parent id
// under a <name>_ref field and points at a list authority. Following all of them
// closes the loop:
//
//	suggestion --search_ref----> search ------> property --reviews_ref----> reviews
//	suggestion --destination---> destination    property --destination_ref-> destination (its city)
//	suggestion --property------> property        review   --property-------> property
//	destination --parent_ref----> destination (up the tree)
//	destination --children_ref--> destinations (down the tree)
//	destination --properties_ref-> properties
//	destination --search_ref-----> search
//
// so a suggestion fans into a search, a place node, or a property; a search card
// walks through to its full property; a property reaches its reviews and the city
// it sits in; a review reaches back to its property; a destination climbs to its
// parent, descends to its children, lists its properties, and fans into a search.
// The geographic tree (country, region, city, district, landmark, airport) and
// the property graph (search, property, reviews) are both fully connected through
// the city node, so a crawl from any seed reaches the rest of the reachable
// estate. No record is a dead leaf: Booking exposes no logged-out reviewer
// profile, so a review points back to its property rather than at a fabricated
// user.

// Property is a Booking.com property, emitted by search (as a card) and by
// property (full detail from the page's JSON-LD island). The id is "<cc>/<slug>",
// the country code and slug in /hotel/<cc>/<slug>.html, so a card and a direct
// read address the same record identically.
type Property struct {
	ID             string   `json:"id" kit:"id"` // "<cc>/<slug>", e.g. "gb/the-savoy"
	Name           string   `json:"name,omitempty" table:",truncate"`
	Type           string   `json:"type,omitempty"`                  // hotel, apartment, hostel, resort, villa, guesthouse, bnb
	Stars          int      `json:"stars,omitempty"`                 // hotel class, 1 to 5
	Rating         float64  `json:"rating,omitempty"`                // review score, 0 to 10
	ReviewCount    int      `json:"review_count,omitempty"`          //
	ReviewWord     string   `json:"review_word,omitempty" table:"-"` // "Superb", "Fabulous", localized
	Price          float64  `json:"price,omitempty"`                 // nightly, only with dates
	Total          float64  `json:"total,omitempty" table:"-"`       // stay total, only with dates
	Currency       string   `json:"currency,omitempty"`              // ISO 4217, when price is set
	Street         string   `json:"street,omitempty" table:"-"`
	City           string   `json:"city,omitempty"`
	Region         string   `json:"region,omitempty" table:"-"`
	Zip            string   `json:"zip,omitempty" table:"-"`
	Country        string   `json:"country,omitempty" table:"-"`
	DisplayAddress []string `json:"display_address,omitempty" table:"-"`
	Lat            float64  `json:"lat,omitempty" table:"-"`
	Lng            float64  `json:"lng,omitempty" table:"-"`
	CheckIn        string   `json:"check_in,omitempty" table:"-"` // property check-in window
	CheckOut       string   `json:"check_out,omitempty" table:"-"`
	Description    string   `json:"description,omitempty" table:",truncate" kit:"body"`
	Amenities      []string `json:"amenities,omitempty" table:"-"`
	Image          string   `json:"image,omitempty" table:",truncate"`
	Photos         []string `json:"photos,omitempty" table:"-"`
	URL            string   `json:"url"`
	ReviewsRef     string   `json:"reviews_ref,omitempty" table:"-" kit:"link,kind=booking/reviews"`         // = ID
	DestinationRef string   `json:"destination_ref,omitempty" table:"-" kit:"link,kind=booking/destination"` // the city node, "city/<cc>/<slug>"
}

// Review is one review of a property, emitted by reviews. Booking splits each
// review into what the guest liked and what they disliked; both are kept, and
// Text is the combined body used by cat and export. Property is the edge back to
// the reviewed property.
type Review struct {
	ID           string  `json:"id" kit:"id"`
	Author       string  `json:"author,omitempty"`
	Country      string  `json:"country,omitempty"` // reviewer's country
	Score        float64 `json:"score,omitempty"`   // 0 to 10
	Date         string  `json:"date,omitempty"`
	Title        string  `json:"title,omitempty" table:",truncate"`
	Positive     string  `json:"positive,omitempty" table:"-"`        // what the guest liked
	Negative     string  `json:"negative,omitempty" table:"-"`        // what the guest disliked
	Text         string  `json:"text,omitempty" table:"-" kit:"body"` // positive + negative, for cat/export
	RoomType     string  `json:"room_type,omitempty" table:"-"`
	Nights       int     `json:"nights,omitempty" table:"-"`
	TravelerType string  `json:"traveler_type,omitempty"` // couple, family, solo, group, business
	Language     string  `json:"language,omitempty" table:"-"`
	Property     string  `json:"property,omitempty" table:"-" kit:"link,kind=booking/property"` // = "<cc>/<slug>"
}

// Destination is one node of Booking's geographic tree, emitted by destination
// (one node) and, as elements, by destinations (a node's children). The id is
// "<kind>/<cc>[/<slug>]", e.g. "country/us", "region/us/florida", "city/us/orlando".
// ChildrenRef and PropertiesRef carry the node's own id so a host follows them
// into the destinations and properties list authorities, and ParentRef climbs the
// tree, so a crawl walks the taxonomy in both directions.
type Destination struct {
	ID            string  `json:"id" kit:"id"`       // "city/us/orlando", "country/us"
	Name          string  `json:"name,omitempty"`    //
	Kind          string  `json:"kind,omitempty"`    // country, region, city, district, landmark, airport
	Country       string  `json:"country,omitempty"` // country name or code
	Region        string  `json:"region,omitempty" table:"-"`
	PropertyCount int     `json:"property_count,omitempty"` // properties Booking lists here
	Lat           float64 `json:"lat,omitempty" table:"-"`
	Lng           float64 `json:"lng,omitempty" table:"-"`
	URL           string  `json:"url"`
	ParentRef     string  `json:"parent_ref,omitempty" table:"-" kit:"link,kind=booking/destination"`    // up the tree
	ChildrenRef   string  `json:"children_ref,omitempty" table:"-" kit:"link,kind=booking/destinations"` // = ID, the child nodes
	PropertiesRef string  `json:"properties_ref,omitempty" table:"-" kit:"link,kind=booking/properties"` // = ID, the properties here
	SearchRef     string  `json:"search_ref,omitempty" table:"-" kit:"link,kind=booking/search"`         // = Name, a free-text search
}

// Suggestion is one autocomplete entry, emitted by suggest. Kind names what the
// suggestion points at; the matching edge is filled accordingly. A place
// suggestion fills Destination and SearchRef; a hotel suggestion fills Property.
// Every suggestion fills SearchRef so a prefix can always fan into a search.
type Suggestion struct {
	Query         string  `json:"query"`          // the prefix that was queried
	Text          string  `json:"text" kit:"id"`  // the suggested label
	Kind          string  `json:"kind,omitempty"` // city, region, district, country, landmark, airport, hotel
	Country       string  `json:"country,omitempty"`
	PropertyCount int     `json:"property_count,omitempty"`    // Booking's hotel count for the match
	DestID        string  `json:"dest_id,omitempty" table:"-"` // Booking's internal destination id
	DestType      string  `json:"dest_type,omitempty" table:"-"`
	Lat           float64 `json:"lat,omitempty" table:"-"`
	Lng           float64 `json:"lng,omitempty" table:"-"`
	SearchRef     string  `json:"search_ref,omitempty" table:"-" kit:"link,kind=booking/search"`       // = Text
	Destination   string  `json:"destination,omitempty" table:"-" kit:"link,kind=booking/destination"` // a place match, "<kind>/<cc>/<slug>"
	Property      string  `json:"property,omitempty" table:"-" kit:"link,kind=booking/property"`       // a hotel match, "<cc>/<slug>"
}

// Ref is the result of `booking ref id`: the canonical (kind, id) a reference
// resolves to, plus the live URL, all without touching the network.
type Ref struct {
	Input string `json:"input"`
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	URL   string `json:"url"`
}
