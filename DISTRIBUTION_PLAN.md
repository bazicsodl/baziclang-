# Distribution & Release Plan

## Packages
- Windows: MSI + ZIP
- macOS: PKG + tar.gz
- Linux: tar.gz

## Signing
- Sign Windows binaries (Authenticode).
- Sign macOS packages (codesign + notarization).
- Provide SHA256 checksums for all artifacts.

## Release Steps
1. Build artifacts for all targets.
2. Run full test suite + benchmarks.
3. Generate SBOM and attach to release.
4. Publish release notes and upgrade guidance.

## Scripts
- Build: `tools/release/build.ps1` / `tools/release/build.sh`
- Package + checksums: `tools/release/package.ps1` / `tools/release/package.sh`
- Sign: `tools/release/sign.ps1` / `tools/release/sign.sh`
- Installers: `tools/release/installer.ps1` (MSI via WiX), `tools/release/installer.sh` (PKG via pkgbuild)
- WiX bootstrap: `tools/release/install_wix.ps1`

## Installer defaults
- Install `bazic`, `bazc`, `bazlsp`, `std/`, `runtime/`.
- Add Bazic to `PATH`.
- Set `BAZIC_HOME` to the install directory.

## Versioned Stdlib
- Tag stdlib snapshots per release.
- Update `std/README.md` with versioned API surface.
