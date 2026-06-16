---
title: "Release notes"
linkTitle: "Release notes"
description: "What changed in each booking release, newest first."
weight: 40
---

What shipped in each release, newest first. Every tagged version builds the same
set of artifacts: archives for Linux, macOS, Windows, and FreeBSD, Linux packages
(deb, rpm, apk), a multi-arch container image on GHCR, and entries for the package
managers. Binaries are pure Go, so there is nothing to install alongside them.

## v0.1.0

The first release. It covers the full read surface over both reliability tiers:
`destination`, `destinations`, and `properties` on the destination estate, and
`property`, `reviews`, `search`, and `suggest` on the interactive client, plus the
offline `ref id` and `ref url` tools. The same operations serve over HTTP
(`serve`) and as an MCP tool set (`mcp`), and the package registers a `booking://`
resource-URI driver.
