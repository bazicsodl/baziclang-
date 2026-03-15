# Bazic Release Process

This document defines the release process for Bazic.

## Versioning policy
- Releases follow `MAJOR.MINOR.PATCH`.
- `MAJOR`: incompatible language or CLI changes.
- `MINOR`: backward-compatible feature additions.
- `PATCH`: bug fixes and performance improvements.

## Stability promise (v1.0)
- v1.0 guarantees stable syntax and semantics as defined in `LANGUAGE.md`.
- Breaking changes are **major‑version only** and must include a migration guide.
- Deprecations require **at least one minor release** before removal.
- Stdlib surface is stable for v1.0, except where marked experimental in docs.

## Channels
- `nightly`: every commit to main.
- `beta`: release candidates.
- `stable`: signed release with changelog and migration notes.

## Release steps
1. Ensure CI gates are green.
2. Build release artifacts: `tools/release/build.ps1` or `tools/release/build.sh`.
3. Package and checksum: `tools/release/package.ps1` or `tools/release/package.sh`.
4. Sign binaries: `tools/release/sign.ps1` (Windows) / `tools/release/sign.sh` (macOS).
5. Build installers: `tools/release/installer.ps1` (MSI) / `tools/release/installer.sh` (PKG).
   - If WiX is not installed, run `tools/release/install_wix.ps1`.
6. Update `LANGUAGE.md` version header if needed.
7. Update `MIGRATIONS.md` with changes since last release.
8. Update `CHANGELOG.md` (or release notes section in README).
9. Tag release: `vX.Y.Z`.
10. Publish binaries and `bazic.lock.json` schema if changed.
11. Publish SBOM and checksums.

## Required artifacts
- `bazc` binary (native + wasm toolchain support).
- `bazic.lock.json` schema version in release notes.
- SBOM (`bazic.sbom.json`).
- Checksums for binaries.

## Backward compatibility
- Only `MAJOR` releases may break source compatibility.
- `MINOR` releases must keep existing code compiling with no changes.
- Deprecations require at least one minor release before removal.
