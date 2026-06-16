---
title: "Output formats"
description: "The output contract every command shares: formats, fields, and templates."
weight: 30
---

Every command renders through one formatter, so the same flags work everywhere.
Pick a format with `-o`, or let booking choose: a table when writing to a
terminal, JSONL when piped.

## Record types

The read commands emit five record types. Each renders through the same formatter,
so the flags below apply to all of them.

- **Property** (`search`, `property`, `properties`): `id` (`<cc>/<slug>`), `name`,
  `type` (hotel, apartment, hostel, resort, villa, guesthouse, bnb), `stars`
  (1-5), `rating` (0-10 review score), `review_count`, `review_word`, `price`
  (nightly, only with dates), `total`, `currency`, `street`, `city`, `region`,
  `zip`, `country`, `display_address[]`, `lat`, `lng`, `check_in`, `check_out`,
  `description`, `amenities[]`, `image`, `photos[]`, `url`, `reviews_ref`,
  `destination_ref`.
- **Review** (`reviews`): `id`, `author`, `country`, `score` (0-10), `date`,
  `title`, `positive`, `negative`, `text`, `room_type`, `nights`,
  `traveler_type`, `language`, `property`.
- **Destination** (`destination`, `destinations`): `id`
  (`<kind>/<cc>[/<slug>]`), `name`, `kind`, `country`, `region`,
  `property_count`, `lat`, `lng`, `url`, `parent_ref`, `children_ref`,
  `properties_ref`, `search_ref`.
- **Suggestion** (`suggest`): `query`, `text`, `kind`, `country`,
  `property_count`, `dest_id`, `dest_type`, `lat`, `lng`, `search_ref`,
  `destination`, `property`.
- **Ref** (`ref id`, `ref url`): `input`, `kind`, `id`, `url`.

## Formats

```bash
booking <command> -o table     # a rounded, color-aware grid for reading
booking <command> -o markdown  # a GitHub pipe table to paste into docs (alias: md)
booking <command> -o list      # one record per section, easy on the eyes
booking <command> -o jsonl     # one JSON object per line, for piping
booking <command> -o json      # a single JSON array
booking <command> -o csv       # spreadsheet friendly
booking <command> -o tsv       # tab-separated
booking <command> -o url       # just the URL column
booking <command> -o raw       # the underlying bytes, unformatted
```

| Format | Best for |
|---|---|
| `table` | Reading on a terminal: a rounded border with an accented header |
| `markdown` | Pasting into a README, issue, or PR (alias `md`) |
| `list` | Reading one record at a time: a heading and a short bullet list per record |
| `jsonl` | Piping into another tool, one object at a time |
| `json` | Loading a whole result as an array |
| `csv` / `tsv` | Spreadsheets and quick column math |
| `url` | Feeding URLs into other commands |
| `raw` | The unformatted bytes (response bodies, file contents) |

## Color

On an interactive terminal the `table`, `list`, and `json`/`jsonl` formats are
colored: the table draws a dim border with an accented header, `list` styles each
record's heading and keys, and JSON keys, strings, numbers, and literals are
highlighted. Color is suppressed the moment output is not a terminal, so a pipe
always gets plain, parseable bytes (and `list` falls back to literal Markdown).
Force the choice with `--color always|never` (or set `NO_COLOR`). `markdown`,
`csv`, `tsv`, `url`, and `raw` are never colored, so they stay safe to redirect
into a file.

## Narrowing columns

Keep only the fields you want:

```bash
booking properties city/us/orlando --fields id,name,stars,rating,price
```

`--no-header` drops the header row in `table` and `csv` output, which helps when
a downstream tool expects bare rows.

## Templating rows

For full control over each line, apply a Go text/template. Fields are the JSON
keys, capitalised:

```bash
booking properties city/us/orlando --template '{{.Name}} {{.Stars}}* {{.Rating}}'
```

## Why auto-detection helps

Because the default adapts to the destination, the same command reads well by
hand and parses cleanly in a pipe:

```bash
booking destinations country/us            # a table, because this is a terminal
booking destinations country/us | wc -l    # JSONL, because this is a pipe
```

You only reach for `-o` when you want something other than that default.
