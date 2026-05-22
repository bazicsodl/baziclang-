# Bazic Stdlib Tiers

This document defines which stdlib modules are part of the Bazic alpha core and which remain experimental.

## Tier 1: Alpha Stable Core

These modules are part of the supported alpha workflow and should be prioritized for compatibility, tests, and docs accuracy.

- `std/io`
- `std/fs`
- `std/time`
- `std/json`
- `std/http`
- `std/crypto`
- `std/base64`
- `std/collections`
- `std/os`
- `std/path`

Expected quality bar for Tier 1:

- documented in `std/README.md`
- covered by compiler or integration tests
- supported on the Go backend release path
- treated as part of the public alpha contract

## Tier 2: Alpha Experimental

These modules exist and may be useful, but they are not part of the stable alpha contract.

- `std/db`
- `std/auth`
- `std/jwt`
- `std/session`
- `std/desktop`
- `std/web`
- `std/ui`
- `std/sql`
- `std/validate`

Experimental means:

- API cleanup is still allowed
- backend support may vary
- docs may lag behind implementation details
- regressions in these modules are lower priority than Tier 1 regressions

## Backend Expectations

- Go backend is the release backend for Tier 1.
- LLVM backend remains experimental even for Tier 1 modules until parity gates are met.
- wasm/web support is best treated as an experiment layered on the Go backend.

## Rule For New APIs

New stdlib APIs should default to experimental unless they are explicitly promoted into Tier 1 with:

- tests
- docs
- backend support expectations
- release-scope justification
