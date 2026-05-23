# Bazic Release Scope

This document defines the Bazic v1 release target.

## Release Objective

Ship Bazic as a compiled, beginner-friendly, statically typed language with a dependable toolchain and a stable core language/runtime for small-to-medium command-line, service, and web-adjacent applications.

## Bazic v1 Must Deliver

- A stable compiler frontend: lexer, parser, semantic analysis, diagnostics.
- A stable release backend: Go backend.
- A functioning package workflow for local and versioned packages.
- A documented and tested core stdlib.
- Reliable tooling: `build`, `run`, `check`, `fmt`, `test`, `lint`.
- A clear installation and upgrade path.
- Compatibility and release policies that match reality.

## Bazic v1 Workloads

- CLI tools
- Small backend services
- JSON/HTTP-oriented applications
- WASM/web experiments through the Go backend

## Bazic v1 Non-Goals

These are explicitly out of scope for initial go-live unless they become blockers:

- Systems-programming-grade ownership/borrow semantics
- Full LLVM-first native release posture
- Production-ready desktop/mobile framework story
- AI/ML runtime and package ecosystem
- Stable FFI/plugin ABI
- A large batteries-included enterprise framework surface

## Backend Policy

- `bazic` release default: Go backend
- `bazc` developer compiler default: Go backend
- LLVM backend status: experimental
- LLVM backend may be used for native smoke tests and backend development, but it is not the primary release contract until parity gates are met

## Standard Library Policy

Stable core for v1:

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
- test helpers and safety prelude

Experimental or lower-confidence areas for v1:

- `std/db`
- `std/auth`
- `std/jwt`
- `std/session`
- `std/desktop`
- `std/web`
- `std/ui`
- `std/sql`
- `std/validate`

See `STDLIB_TIERS.md` for the maintained alpha stable-core split and `RUNTIME_CONTRACT.md` for the broader compiler/runtime/package boundary.

## Release Quality Bar

Bazic v1 is not ready until all of the following are true:

- `go test ./...` passes
- Conformance suite passes on the Go backend
- Public docs match actual syntax and behavior
- Package verification and lockfile workflow are stable
- Diagnostics include exact source locations
- Installation quickstart works on supported platforms
- No open compiler crashers in the supported core workflow

## Supported Release Posture

For Bazic v1, the project should communicate:

- stable frontend
- stable Go backend
- experimental LLVM backend
- selective stdlib stability, not blanket ecosystem maturity

## Decision Rule

If a feature competes with compiler architecture, diagnostics, backend stability, or packaging reliability, the feature loses until those core items are done.
