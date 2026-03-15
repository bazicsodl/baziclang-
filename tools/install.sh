#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT_DIR"

OUT_DIR="${1:-./bin}"
mkdir -p "$OUT_DIR"

go build ./cmd/bazic -o "$OUT_DIR/bazic"
go build ./cmd/bazc -o "$OUT_DIR/bazc"
go build ./cmd/bazlsp -o "$OUT_DIR/bazlsp"

if [ -d "$ROOT_DIR/std" ]; then
  rm -rf "$OUT_DIR/std"
  cp -R "$ROOT_DIR/std" "$OUT_DIR/std"
  echo "Copied stdlib to $OUT_DIR/std"
fi

if [ -d "$ROOT_DIR/runtime" ]; then
  rm -rf "$OUT_DIR/runtime"
  cp -R "$ROOT_DIR/runtime" "$OUT_DIR/runtime"
  echo "Copied runtime to $OUT_DIR/runtime"
fi

VSIX_SRC="$ROOT_DIR/tools/vscode/baziclang-0.1.0.vsix"
VSIX_OUT="$OUT_DIR/baziclang.vsix"
if [ -f "$VSIX_SRC" ]; then
  cp "$VSIX_SRC" "$VSIX_OUT"
  echo "Copied VS Code extension to $VSIX_OUT"
  if command -v code >/dev/null 2>&1; then
    code --install-extension "$VSIX_OUT" --force >/dev/null 2>&1 || true
    echo "Installed Bazic VS Code extension"
  fi
fi

echo "Built $OUT_DIR/bazic"
echo "Built $OUT_DIR/bazc"
echo "Built $OUT_DIR/bazlsp"
