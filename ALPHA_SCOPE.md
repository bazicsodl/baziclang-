# Bazic Alpha Scope

This document defines the Bazic alpha release contract.

## Alpha Objective

Ship a usable Bazic toolchain for:

- small CLI applications
- small JSON/HTTP services
- package-managed local projects
- formatter/check/test/lint driven development

Alpha is the first public posture where Bazic should feel coherent to an external user, even if the overall language and ecosystem are not feature-complete.

## Supported Alpha Toolchain

- Stable default backend: Go
- Experimental backend: LLVM
- Supported commands:
  - `bazic new`
  - `bazic init`
  - `bazic build`
  - `bazic run`
  - `bazic check`
  - `bazic fmt`
  - `bazic test`
  - `bazic lint`
  - `bazic doctor`
  - `bazic pkg init|add|sync|verify|sbom`

## Supported Alpha Language Surface

- functions
- `let`, `const`
- `if`, `else`, `while`, `return`
- structs and struct literals
- enums
- interfaces and `impl`
- generic structs and generic functions
- package declarations
- package imports with aliases
- `pub` visibility
- exhaustive enum `match`
- `Option`, `Result`, `Error`
- builtin string helpers and parse helpers

## Supported Alpha Workloads

- single-binary CLI tools
- local package-based apps
- simple service processes using `std/http`
- wasm/web experiments through the Go backend

## Explicit Alpha Non-Goals

- stable native LLVM release posture
- production-grade database framework surface
- production-grade auth/session stack
- desktop/mobile application promise
- stable UI/web framework contract
- stable plugin/FFI ABI
- systems-programming memory model claims

## Alpha Standard Library Tiers

Alpha stable core:

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

Alpha experimental surface:

- `std/db`
- `std/auth`
- `std/jwt`
- `std/session`
- `std/desktop`
- `std/web`
- `std/ui`
- `std/sql`
- `std/validate`

See `STDLIB_TIERS.md` for the detailed policy.

## Alpha Exit Criteria

- `go test ./...` passes
- Go backend is the documented and tested default path
- `bazic new`, `build`, `run`, `check`, `fmt`, `test`, `lint`, and `doctor` work end-to-end
- docs match the implemented grammar and package rules
- stdlib tiers are documented and reflected in public messaging
- at least three reference apps build and run on the Go backend
- no known compiler crashers in the supported alpha workflow

## Messaging Rule

When Bazic talks about alpha publicly, it should say:

- stable alpha frontend/toolchain on the Go backend
- experimental LLVM backend
- stable alpha stdlib core, not a blanket stable ecosystem
