package booking

import (
	"context"

	"github.com/tamnd/any-cli/kit/errs"
)

// ops.go holds the handler for every operation declared in domain.go. kit
// reflects each input struct into CLI flags, HTTP query params, and MCP tool
// arguments: kit:"arg" is a positional, kit:"flag,inherit" binds the shared
// --limit, and kit:"inject" receives the client newClient builds. The reference
// ops (id, url) take no client; they run offline.

// defaultLimit is the fetch count an op falls back to when --limit is unset, so a
// bare command returns a useful page without flooding a terminal.
const defaultLimit = 20

// --- search ---

type searchIn struct {
	Destination string  `kit:"arg" help:"a place to search, e.g. \"Orlando\""`
	Limit       int     `kit:"flag,inherit"`
	Client      *Client `kit:"inject"`
}

func search(ctx context.Context, in searchIn, emit func(*Property) error) error {
	items, err := in.Client.Search(ctx, in.Destination, limitOr(in.Limit, defaultLimit))
	if err != nil {
		return mapErr(err)
	}
	return emitAll(items, emit)
}

// --- property ---

type propertyIn struct {
	Ref    string  `kit:"arg" help:"property id \"<cc>/<slug>\" or /hotel/ URL"`
	Client *Client `kit:"inject"`
}

func getProperty(ctx context.Context, in propertyIn, emit func(*Property) error) error {
	p, err := in.Client.GetProperty(ctx, in.Ref)
	if err != nil {
		return mapErr(err)
	}
	return emit(p)
}

// --- reviews ---

type reviewsIn struct {
	Ref    string  `kit:"arg" help:"property id \"<cc>/<slug>\" or /hotel/ URL"`
	Limit  int     `kit:"flag,inherit"`
	Client *Client `kit:"inject"`
}

func reviews(ctx context.Context, in reviewsIn, emit func(*Review) error) error {
	rs, err := in.Client.Reviews(ctx, in.Ref, limitOr(in.Limit, 0))
	if err != nil {
		return mapErr(err)
	}
	return emitAll(rs, emit)
}

// --- destination ---

type destinationIn struct {
	Ref    string  `kit:"arg" help:"destination ref \"<kind>/<cc>[/<slug>]\" or landing URL"`
	Client *Client `kit:"inject"`
}

func getDestination(ctx context.Context, in destinationIn, emit func(*Destination) error) error {
	d, err := in.Client.GetDestination(ctx, in.Ref)
	if err != nil {
		return mapErr(err)
	}
	return emit(d)
}

// --- destinations (children of a node) ---

type destListIn struct {
	Ref    string  `kit:"arg" help:"destination ref or landing URL"`
	Limit  int     `kit:"flag,inherit"`
	Client *Client `kit:"inject"`
}

func listDestinations(ctx context.Context, in destListIn, emit func(*Destination) error) error {
	ds, err := in.Client.ListChildren(ctx, in.Ref, limitOr(in.Limit, 0))
	if err != nil {
		return mapErr(err)
	}
	return emitAll(ds, emit)
}

// --- properties (on a destination landing page) ---

type propsListIn struct {
	Ref    string  `kit:"arg" help:"destination ref or landing URL"`
	Limit  int     `kit:"flag,inherit"`
	Client *Client `kit:"inject"`
}

func listProperties(ctx context.Context, in propsListIn, emit func(*Property) error) error {
	ps, err := in.Client.ListProperties(ctx, in.Ref, limitOr(in.Limit, 0))
	if err != nil {
		return mapErr(err)
	}
	return emitAll(ps, emit)
}

// --- suggest ---

type prefixIn struct {
	Prefix string  `kit:"arg" help:"the typed prefix"`
	Limit  int     `kit:"flag,inherit"`
	Client *Client `kit:"inject"`
}

func suggest(ctx context.Context, in prefixIn, emit func(*Suggestion) error) error {
	ss, err := in.Client.Suggest(ctx, in.Prefix, limitOr(in.Limit, 0))
	if err != nil {
		return mapErr(err)
	}
	return emitAll(ss, emit)
}

// --- sitemap (the crawl root) ---

type sitemapIn struct {
	Kind   string  `kit:"arg" help:"country, region, city, district, landmark, airport, or hotel"`
	Limit  int     `kit:"flag,inherit"`
	Client *Client `kit:"inject"`
}

func sitemap(ctx context.Context, in sitemapIn, emit func(*Seed) error) error {
	seeds, err := in.Client.Sitemap(ctx, in.Kind, limitOr(in.Limit, defaultLimit))
	if err != nil {
		return mapErr(err)
	}
	return emitAll(seeds, emit)
}

// --- reference tools (offline) ---

type refIn struct {
	Ref string `kit:"arg" help:"any Booking.com URL, path, or id"`
}

func classifyRef(_ context.Context, in refIn, emit func(*Ref) error) error {
	r := Classify(in.Ref)
	if r.Kind == "unknown" {
		return errs.Usage("unrecognized booking reference: %q", in.Ref)
	}
	return emit(&r)
}

type urlIn struct {
	Kind string `kit:"arg" help:"property, destination, or search"`
	ID   string `kit:"arg" help:"the id for that kind"`
}

func buildURL(_ context.Context, in urlIn, emit func(*Ref) error) error {
	u := URLFor(in.Kind, in.ID)
	if u == "" {
		return errs.Usage("booking has no resource type %q", in.Kind)
	}
	return emit(&Ref{Input: in.Kind + "/" + in.ID, Kind: in.Kind, ID: in.ID, URL: u})
}

// emitAll streams a slice of records through emit.
func emitAll[T any](items []*T, emit func(*T) error) error {
	for _, it := range items {
		if err := emit(it); err != nil {
			return err
		}
	}
	return nil
}

// limitOr returns the operator's --limit when set, else the command's own default
// fetch count.
func limitOr(limit, def int) int {
	if limit > 0 {
		return limit
	}
	return def
}
