# Bazic v1.0.0 Publish Checklist

1. Windows signing
1. If no cert yet: mark release as **unsigned** and publish `SHA256SUMS.txt`.
1. If signing: ensure `BAZIC_WIN_CERT_PFX` and `BAZIC_WIN_CERT_PASS` are set.
1. If signing: run `tools/release/sign.ps1`.
1. If signing: verify signed binaries in `dist/release/windows-amd64`.
1. If signing: verify `dist/packages/bazic-windows-amd64.msi` is signed.
1. If signing: rebuild MSI if required by your signing policy.

1. macOS signing and notarization (macOS host)
1. Ensure `BAZIC_MAC_IDENTITY`, `BAZIC_APPLE_ID`, `BAZIC_APPLE_PASS`, `BAZIC_MAC_TEAM_ID` are set.
1. Run `tools/release/sign.sh`.
1. Run `tools/release/installer.sh` to build PKG.

1. Linux baseline capture (if required by CI policy)
1. Run `scripts/bench_baseline.ps1` or `scripts/bench_baseline.sh` on Linux host.

1. macOS baseline capture (if required by CI policy)
1. Run `scripts/bench_baseline.sh` on macOS host.

1. Release artifacts
1. Confirm `dist/packages` contains: zip/tar.gz, MSI, `SHA256SUMS.txt`, `RELEASE_MANIFEST.txt`.
1. Confirm `dist/release/bazic.sbom.json` exists.
1. Confirm `dist/packages/VERIFY_CHECKSUMS_WINDOWS.txt` exists for end users.

1. Tag and publish
1. Tag `v1.0.0`.
1. Upload artifacts from `dist/packages`.
1. Publish `CHANGELOG.md` and `CHANGELOG.json`.
