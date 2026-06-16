---
title: "Troubleshooting"
description: "The handful of things that trip people up, and how to fix each one."
weight: 40
---

Most of these come down to network reality or how Booking.com serves its data.
booking maps each outcome to a stable exit code, so a script can tell them apart.

## Exit codes

| Code | Name | Meaning |
|---|---|---|
| 0 | ok | Success |
| 1 | generic | An unclassified error |
| 2 | usage | Bad flags or arguments |
| 3 | no results | The query ran but matched nothing |
| 4 | need-auth | The bot wall on the best-effort tier |
| 5 | rate limited | Booking.com is asking you to slow down |
| 6 | not found | The reference points at nothing |
| 7 | unsupported | The operation cannot serve this input |
| 8 | network | A transport failure reaching the site |

## Exit 4: the bot wall

`property`, `reviews`, `search`, and `suggest` read the interactive client, which
sits behind a bot manager. From a datacenter that manager often returns a wall,
and booking reports it as exit code 4 (need-auth).

There are two remedies:

- Retry from a residential or mobile connection, where the interactive client
  works.
- Read the destination estate instead. The `destination`, `destinations`, and
  `properties` commands read the country, region, city, district, landmark, and
  airport landing pages, which exist to be crawled and are not gated. They read
  from anywhere, so they are the reliable fallback. You can often reach the same
  property through `properties` on its city node rather than through `property`
  directly.

## Exit 5: rate limited

Booking.com rate-limits like any public site. booking already paces requests and
retries the transient failures, but a hard limit still means backing off. Raise
the delay between requests with `--rate` (for example `--rate 1s`) and retry
later. A burst of 429 or 5xx responses is the site asking you to slow down.

## Exit 6: not found

The reference points at nothing that exists. Check that the input is spelled the
way Booking.com uses it: a property id is `<cc>/<slug>` (for example
`gb/the-savoy`), and a destination ref is `<kind>/<cc>[/<slug>]` (for example
`city/us/orlando`). Use `booking ref id <url>` to classify a pasted URL into its
(kind, id) before passing it on.

## A price came back empty

A nightly price is filled only when you pass both `--checkin` and `--checkout`.
With one or neither, the `price` and `total` fields stay empty by design. booking
never invents a price.

## The binary is not on your PATH

`go install` puts the binary in `$(go env GOPATH)/bin` (usually `~/go/bin`), and
a release archive leaves it wherever you unpacked it. If your shell cannot find
`booking`, add that directory to your `PATH`. See
[installation](/getting-started/installation/).

## Seeing what booking actually did

When something behaves unexpectedly, `-v` adds per-request detail so you can see
the URLs it hit and the responses it got. That is usually enough to tell a bot
wall apart from a rate limit or a genuinely empty result.
