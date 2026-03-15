# Performance & Optimization Plan

This document defines how we will reach production-grade performance in Bazic.

## Goals
- Keep Go backend as reference baseline.
- Close LLVM performance gaps for core benchmarks.
- Ensure no regressions in hot paths (parser, codegen, runtime, stdlib).

## Phase 3 Checklist
1. Profiling
   - Add compiler profiling hooks for codegen hot paths.
   - Profile runtime benchmarks under Go and LLVM backends.
2. Optimizations
   - Reduce allocations in string ops and JSON routines.
   - Optimize database row mapping hot paths.
   - Improve codegen for string concatenation and comparisons.
3. Regression Guards
   - Enforce bench gates in CI for Go and LLVM.

## Initial Focus Areas
- `internal/codegen/go.go`: string concat/formatting, JSON helpers.
- `runtime/bazic_runtime.c`: JSON parsing, base64, crypto.
- `internal/compiler`: build/run pipeline overhead.

## Completed
- Optimization pass 1: reduced allocations in header/kv parsing (Go backend).
- Optimization pass 2: streaming JSON key lookup in Go backend (reduced allocations).
- Optimization pass 3: stream JSON row encoding to avoid large in-memory slices.

## Tooling
- `go test -bench` for compiler benchmarks.
- `scripts/bench.ps1` / `bench_gate.ps1` for runtime benchmarks.
- `bench-baseline` GitHub workflow captures weekly baselines.
- Optional: `pprof` for Go backend.

## Completion Criteria
- Bench targets in `BENCHMARKS.md` met or improved.
- No regressions across 3 consecutive baseline captures.
- Documented perf deltas with summary in release notes.
