# Bazic Safety Model (v1.0)

This document defines the safety posture for Bazic codebases and the compiler/runtime behavior expected for v1.0.

## 1. Core Safety Principles
- **No implicit `nil`**: `nil` is rejected at compile time. Use `Option[T]`, `Result[T,E]`, or `Error`.
- **Explicit error flow**: Functions that can fail should return `Result[T,E]` (or `Option[T]` when the absence of a value is not an error).
- **No silent type coercion**: All conversions must be explicit; `any` does not auto‑convert.
- **Deterministic control flow**: `match` is exhaustive for enums.
- **No implicit globals**: All globals must be declared.

## 2. `any` Usage Policy
`any` is intended only for **interop boundaries** (I/O, logging, dynamic data edges).

Allowed:
- Logging/printing (`print`, `println`, `str`).
- Boundary layers: JSON parsing outputs, FFI payloads, plugin interfaces.

Disallowed in core domain logic:
- Struct fields typed as `any`.
- Function parameters typed as `any`.
- Collections containing `any`.

Lint rule `BL006` enforces this policy outside conformance/stdlib code.

## 3. Error Flow Guidelines
Prefer:
- `Result[T,E]` for fallible APIs.
- `Option[T]` for presence/absence.

Avoid:
- Using sentinel values (`""`, `0`, `false`) to encode errors.
- Returning `any` to bypass type checks.

## 4. Unsafe/FFI Boundary (Future)
Bazic does **not** expose unsafe operations yet. When FFI is introduced:
- All FFI must be **opt‑in** and explicitly marked.
- Unsafe operations must be isolated and audited.
- Tooling will require a safety declaration (`unsafe` module header or similar).

## 5. Compiler/Lint Guarantees
Current enforcement:
- Compile‑time rejection of `nil`.
- Exhaustive `match` over enums.
- No unused variables (forces intentional discard via `_`).
- Lint checks (`bazc lint`) for unsafe patterns and internal APIs.

Planned:
- Explicit unsafe annotation and audit gates.
- Stronger linting for `any` sprawl.
- Verified boundary checks for FFI and plugin models.
