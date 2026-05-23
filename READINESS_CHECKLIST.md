# v1.0 Readiness Checklist

This checklist tracks the final gates for Bazic v1.0.

Important: alpha and v1 are not the same thing. Items below should only be marked complete when they are enforced or verified by the real release workflow, not because a document exists.

## Language & Spec
- [x] `LANGUAGE.md` marked stable (no Draft label)
- [x] Compatibility policy enforced in CI
- [x] `V1_STABILITY.md` finalized

## Tooling
- [x] `bazic` and `bazc` binaries packaged for Windows/macOS/Linux
- [x] LSP + VS Code extension release

## Backends
- [x] LLVM conformance passes on all CI targets
- [x] `emit-llvm --check` gate green

## Performance
- [x] Bench baselines recorded for Windows, Linux, macOS
  - [x] Windows baseline recorded (`bench/baseline.windows.xml`)
  - [x] Linux baseline recorded (`bench/baseline.linux.xml`)
  - [x] macOS baseline recorded (`bench/baseline.macos.xml`)
- [x] Regression gates enabled

## Safety
- [x] `SAFETY.md` finalized
- [x] `any` lint policy enforced
- [x] Unsafe/FFI policy defined

## Docs
- [x] `GETTING_STARTED.md` complete
- [x] `V1_GUIDE.md` finalized
- [x] `MIGRATIONS.md` up to date
- [x] `COMPATIBILITY_MATRIX.md` finalized

## Reference Apps
- [x] Alpha-gated CLI app documented
- [x] Alpha-gated service app documented
- [x] Alpha-gated web/wasm demo documented
- [x] Alpha-gated reference apps enforced in CI/release workflow
- [x] Experimental demos clearly separated from the supported alpha contract
