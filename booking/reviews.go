package booking

import (
	"context"
	"strconv"
)

// reviews.go reads a property's reviews from the review entries embedded in its
// page's JSON-LD island. Booking renders a set of recent reviews into the island
// as schema.org Review objects; booking-cli maps those into Review records. Like
// the property page, this is the best-effort tier, and a walled fetch returns
// ErrBlocked before the parser runs.

// Reviews returns up to limit reviews for a property by id ("<cc>/<slug>") or URL.
func (c *Client) Reviews(ctx context.Context, ref string, limit int) ([]*Review, error) {
	id := propertyID(ref)
	if id == "" {
		return nil, ErrUsage
	}
	body, err := c.get(ctx, URLFor("property", id))
	if err != nil {
		return nil, err
	}
	doc := hotelDoc(body)
	if doc == nil {
		return nil, nil
	}
	var out []*Review
	for i, r := range doc.Review {
		text := squish(r.ReviewBody)
		rv := &Review{
			ID:       id + ":" + strconv.Itoa(i+1),
			Author:   squish(r.Author.Name),
			Title:    squish(r.Name),
			Date:     squish(r.DatePublished),
			Score:    r.ReviewRating.RatingValue.float(),
			Positive: text,
			Text:     text,
			Language: squish(r.InLanguage),
			Property: id,
		}
		if rv.Author == "" && rv.Text == "" {
			continue
		}
		out = append(out, rv)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}
