# Release Checklist (Marketplace)

1. Update `package.json`
1. Set `version` to the next release
1. Ensure `icon` exists at `images/icon.png`
1. Verify `README.md` and `LICENSE` are present

## Build VSIX (no vsce required)
1. Ensure `.vsix-build/` is clean
1. Copy extension files into `.vsix-build/extension`
1. Ensure `.vsix-build/extension.vsixmanifest` and `.vsix-build/[Content_Types].xml` exist
1. Zip `.vsix-build/*` to `baziclang-<version>.zip`
1. Rename to `baziclang-<version>.vsix`

## Publish
1. Go to the Marketplace publisher portal
1. Upload the `.vsix`
