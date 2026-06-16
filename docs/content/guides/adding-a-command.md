---
title: "Add a command"
description: "Model a Booking.com record and expose it as a command, a route, and a tool at once."
weight: 10
---

booking already models the records Booking.com serves: Property, Review,
Destination, Suggestion, and Ref. When you add a new surface, you do it in two
files, and every other surface updates itself.

## 1. Model the record

In `booking/booking.go`, add a struct for the thing you are fetching and a client
method that returns it. The `kit` struct tags decide how a host addresses the
record. The Property type is the pattern to follow:

```go
type Property struct {
    ID           string   `json:"id"            kit:"id"`   // "<cc>/<slug>"
    Name         string   `json:"name"`
    Type         string   `json:"type"`                     // hotel, apartment, ...
    Stars        int      `json:"stars"`
    Rating       float64  `json:"rating"`                   // 0-10 review score
    Description  string   `json:"description"   kit:"body"` // what cat and Markdown print
    ReviewsRef   string   `json:"reviews_ref"   kit:"link,kind=booking/reviews"`
    DestRef      string   `json:"destination_ref" kit:"link,kind=booking/destination"`
    URL          string   `json:"url"`
}

func (c *Client) GetProperty(ctx context.Context, ref string) (*Property, error) {
    body, err := c.Get(ctx, buildHotelURL(ref))
    if err != nil {
        return nil, err
    }
    // parse the page's JSON-LD island into a Property ...
    return prop, nil
}
```

- `kit:"id"` marks the field that becomes the URI id.
- `kit:"body"` marks the prose that `cat` and the Markdown export render.
- `kit:"link,kind=<scheme>/<type>"` marks an outbound edge. It can point at
  another booking type (here `reviews` and `destination`) or at another site
  entirely, which is what lets a host walk the graph across tools.

## 2. Declare the operation

In `booking/domain.go`, add an input struct and a handler, then register it in
`Register`:

```go
type propertyRef struct {
    Ref    string  `kit:"arg" help:"property id (<cc>/<slug>) or /hotel/ URL"`
    Client *Client `kit:"inject"`
}

func getProperty(ctx context.Context, in propertyRef, emit func(*Property) error) error {
    p, err := in.Client.GetProperty(ctx, in.Ref)
    if err != nil {
        return mapErr(err)
    }
    return emit(p)
}

// inside Register(app):
kit.Handle(app, kit.OpMeta{Name: "property", Group: "read", Single: true,
    Summary: "Show one property by id or URL", URIType: "property", Resolver: true,
    Args: []kit.Arg{{Name: "ref", Help: "property id (<cc>/<slug>) or /hotel/ URL"}}}, getProperty)
```

That is the whole change. `kit.Handle` reflects the input for flags and the output
for the record shape, so the operation immediately becomes:

```bash
booking property gb/the-savoy                     # the command
curl 'localhost:7777/v1/property/gb/the-savoy'    # the route, under serve
ant get booking://property/gb/the-savoy           # the URI dereference, via a host
```

## Resolver ops and list ops

Two flags shape how a host treats an operation:

- **`Single: true`** with **`Resolver: true`** marks the canonical one-record
  fetch for a `URIType`. It answers `ant get`. `property`, `destination`, and the
  offline `ref` ops are built this way.
- **`List: true`** marks a member-lister for a parent resource. It answers
  `ant ls`. A list op should emit records that are themselves addressable, so
  every member is a URI a host can follow. The `properties` op does this by
  emitting Property cards, and `destinations` does it by emitting child
  Destination nodes.

## Map errors to exit codes

Return the `errs` kinds from `mapErr` so every surface reports the same outcome
with the same exit code. The bot wall on the best-effort tier maps to need-auth,
which is exit code 4:

```go
case errors.Is(err, ErrBotWall):
    return errs.NeedAuth("%s", err.Error())
case errors.Is(err, ErrNotFound):
    return errs.NotFound("%s", err.Error())
case errors.Is(err, ErrRateLimited):
    return errs.RateLimited("%s", err.Error())
```

See [output formats](/reference/output/) for how records render, and
[resource URIs](/guides/resource-uris/) for how a host addresses them.
