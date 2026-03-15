# Production Readiness Plan

This document tracks what remains to reach production-grade readiness comparable to mainstream languages.

## Phase 1: Stability + Spec
- Write v1 language spec (syntax, types, runtime behavior, stdlib contracts).
- Define backward-compatibility policy and deprecation process.
- Freeze stdlib public APIs for v1.
- Formalize target matrix (Go backend, LLVM native, WASM) and supported OS versions.

## Phase 2: Security Hardening
- Audit stdlib (crypto, http, db, json) for unsafe defaults.
- Add fuzzing for parser + HTTP + JSON routines.
- Document secure defaults and configuration environment variables.
- Establish a security advisory workflow and patch cadence.

## Phase 3: Performance + Reliability
- Benchmark suite for compiler + runtime + stdlib.
- Optimize hot paths in codegen and runtime.
- Ensure deterministic builds for release artifacts.
- Add integration tests for Go/LLVM parity.

## Phase 4: Tooling + DX
- Improve compiler error messages and source mapping.
- Add stack traces for runtime panics in generated binaries.
- Enhance CLI docs and quickstart guides.
- Add full-stack starter templates.

## Phase 5: Release + Distribution
- Windows/macOS/Linux installers.
- Versioned standard library release artifacts.
- Signed binaries and checksums.

## Status (2026-03-10)
- Core stack implemented: routing, DB, migrations, models, auth, JWT, OpenAPI, UI/WASM.
- HTTP server hardened with defaults + env overrides.
- Lexer/parser fuzz tests added.
- HTTP handler parsing and model schema JSON fuzz tests added.
- Go test suite passes.
- Stdlib security guidance documented.
- Bench suite updated with JWT sign/verify and fresh baseline captured.
- Go/LLVM parity tests added for base64/crypto/jwt/json/session/http helpers.
- DB parity test added (guarded by BAZIC_SQLITE_TEST=1).
- Performance and distribution plans documented.
- Optimization pass 1: header/kv parsing in Go backend reduced allocations.
- Optimization pass 2: streaming JSON key lookup in Go backend.
- Optimization pass 3: stream DB JSON rows to avoid large slices.
