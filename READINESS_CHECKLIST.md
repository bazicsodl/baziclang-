# v1.0 Readiness Checklist

This checklist tracks the final gates for v1.0.

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
- [ ] Bench baselines recorded for Windows, Linux, macOS
  - [x] Windows baseline recorded (`bench/baseline.xml`)
  - [ ] Linux baseline recorded (`bench/baseline.xml`)
  - [ ] macOS baseline recorded (`bench/baseline.xml`)
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
- [x] CLI app documented
- [x] Service app documented
- [x] Desktop app documented
- [x] Web/WASM demo documented
