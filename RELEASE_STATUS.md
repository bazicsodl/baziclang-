# Release Status (v1.0)

## Completed
- Build artifacts: `dist/release`
- Packages: `dist/packages` (zip/tar.gz) + `SHA256SUMS.txt`
- Release manifest: `dist/packages/RELEASE_MANIFEST.txt`
- SBOM: `dist/release/bazic.sbom.json`
- Release notes: `CHANGELOG.md`, `CHANGELOG.json`, `MIGRATIONS.md`
- Release tooling: `tools/release/*` (build, package, sign, installer)
- MSI installer: `dist/packages/bazic-windows-amd64.msi`

## Pending (Requires macOS host)
- Codesign + notarization:
  - `tools/release/sign.sh`
  - Env: `BAZIC_MAC_IDENTITY`, `BAZIC_APPLE_ID`, `BAZIC_APPLE_PASS`, `BAZIC_MAC_TEAM_ID`
- PKG installers:
  - `tools/release/installer.sh` (requires `pkgbuild`)

## Pending (Requires Windows signing cert)
- Authenticode signing:
  - `tools/release/sign.ps1`
  - Env: `BAZIC_WIN_CERT_PFX`, `BAZIC_WIN_CERT_PASS`
  - If no cert: mark release as unsigned and publish SHA256SUMS.

## Pending (Requires WiX download)
None.

## Tag/Publish
- Tag `v1.0.0` and publish artifacts.
