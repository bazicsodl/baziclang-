# Bazic Runtime Contract

This document defines the maintained boundary between the Bazic compiler, the Bazic runtime, and the Bazic standard library.

The goal is to keep the supported surface explicit:

- what the language guarantees implicitly
- what the runtime is allowed to implement behind those guarantees
- what the stdlib exposes as package-level APIs

## Compiler-Owned Implicit Surface

These are language-level surfaces the compiler owns directly.

- Safety prelude types:
  - `Error`
  - `Option[T]`
  - `Result[T,E]`
- Safety prelude helpers:
  - `some`
  - `none`
  - `ok`
  - `err`
  - `unwrap_or`
  - `result_or`
  - `assert`
  - `assert_msg`
- Builtin function surface:
  - `print`
  - `println`
  - `str`
  - `len`
  - `contains`
  - `starts_with`
  - `ends_with`
  - `to_upper`
  - `to_lower`
  - `trim_space`
  - `replace`
  - `repeat`
  - `parse_int`
  - `parse_float`

Rules:

- Source compatibility and diagnostics for this implicit surface are compiler responsibilities.
- User programs should not depend on backend-specific runtime symbol names.
- Adding or removing implicit builtins/prelude helpers is a language-contract change, not an implementation-only change.

## Runtime-Owned Implementation Surface

These are implementation details that exist to realize the compiler-owned surface and selected stdlib behavior.

- LLVM runtime C implementation:
  - `runtime/bazic_runtime.c`
- Shared runtime symbol surface:
  - centralized in `internal/intrinsics/runtime_symbols.go`
- Shared runtime/backend selection metadata:
  - `internal/backendmeta/runtime.go`
  - `internal/backendmeta/runtime_artifacts.go`
  - `internal/backendmeta/program_shape.go`
- Shared runtime interface/backend surfaces:
  - `internal/intrinsics/runtime_surface.go`

Rules:

- Runtime symbols, section ordering, helper registration, and backend-specific declarations are implementation surfaces, not user-facing API.
- The runtime may differ internally between Go and LLVM backends as long as the compiler-owned implicit surface and documented stdlib behavior remain intact.
- No stable plugin ABI or stable FFI ABI is promised in the current alpha/release posture.

## Stdlib-Owned Package Surface

These are package APIs exposed through `import "std";`.

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

Rules:

- The alpha stable core is part of the supported public release posture on the Go backend.
- Experimental modules may change faster and are not equal to the supported alpha core.
- Promoting a module from experimental to stable requires matching docs, tests, backend expectations, and release-scope justification.

## Backend And Stability Policy

- Go backend: stable release path
- LLVM backend: experimental
- Go/WASM is supported as an experiment layered on the Go backend, not as a separate stable backend contract.

Implications:

- Bazic release messaging should describe the Go backend plus the alpha stable stdlib core as the supported path.
- LLVM remains opt-in and may be used for native smoke tests, conformance, benchmarking, and backend development without implying stable release posture.

## Drift Rule

Any change to one of these surfaces must update the corresponding maintained contract:

- compiler-owned implicit surface
- runtime/backend registry surface
- stdlib stability tier surface

At minimum, changes in this area should stay aligned across:

- `RUNTIME_CONTRACT.md`
- `ALPHA_SCOPE.md`
- `STDLIB_TIERS.md`
- `RELEASE_SCOPE.md`
- `README.md`
- `internal/releasecontract`
