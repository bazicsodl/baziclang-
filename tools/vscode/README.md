# Bazic VS Code Extension

Language support for Bazic:
- Syntax highlighting
- Auto-closing brackets/quotes
- LSP: diagnostics, completion, go-to-def, rename, formatting, quick fixes
- Auto-fix on save (missing semicolons/quotes/brackets, invalid escapes, &&/||) and format on save

## Quick start
1. Install dependencies in `tools/vscode`:
   - `npm install`
2. Launch the extension in VS Code (Run Extension).
3. Open a `.bz` file. Diagnostics, completion, go-to-def, and rename should be active.

The language server is started via:
- `bazlsp` from PATH (preferred)
- Falls back to local `bin/bazlsp.exe` if present

Make sure Go is available in your PATH.

## Settings
- `bazic.autoFixOnSave` (default: true)
- `bazic.formatOnSave` (default: true)

## Publishing
- Bump version in `package.json`
- Build VSIX (see `RELEASE.md`)
- Upload the VSIX to the Marketplace
