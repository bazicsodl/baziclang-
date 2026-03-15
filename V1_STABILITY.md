# Bazic v1.0 Stability Promise

This document defines the stability guarantees for Bazic v1.x.

## Versioning
- Bazic follows **Semantic Versioning**: `MAJOR.MINOR.PATCH`.
- **MAJOR**: breaking changes.
- **MINOR**: new features, no breaking changes.
- **PATCH**: bug fixes and security fixes only.

## Compatibility Policy
### Source Compatibility (v1.x)
- No breaking language changes in `v1.MINOR`.
- Deprecations require:
  - A clear warning in release notes.
  - A migration note or tool guidance.
  - At least one full minor release before removal.

### Tooling Compatibility
- `bazic` and `bazc` maintain CLI flag stability within v1.x.
- Any flag removal requires a deprecation cycle.

### Stdlib Compatibility
- Public `std` APIs are stable in v1.x.
- Additive APIs are allowed in minors.
- Removed or changed APIs require deprecation + migration notes.

## Security & Patch Policy
- Security fixes can be shipped in PATCH releases.
- Security fixes will not introduce breaking behavior unless unavoidable.

## Migration Policy
- Every breaking change includes:
  - A migration entry in `MIGRATIONS.md`.
  - If feasible, an automated migration note in `MIGRATION_TOOLING.md`.

## Supported Targets
- Go backend: Windows/Linux/macOS.
- LLVM backend: Windows/Linux/macOS.
- WASM target (browser) supported for Go backend.

## Deprecation Schedule
- Minimum: 1 minor release between deprecation and removal.
- Prefer 2 minors for high‑impact changes.
