#!/usr/bin/env bash
set -euo pipefail

OUT_DIR="${1:-dist/desktop}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

SRC="examples/apps/desktop/main.bz"
BIN_DIR="$OUT_DIR/bin"
mkdir -p "$BIN_DIR"

EXE="$BIN_DIR/bazic-desktop"
go run ./cmd/bazc build "$SRC" -o "$EXE"

cat > "$OUT_DIR/README.txt" <<'EOF'
Bazic Desktop MVP

Run:
  ./bin/bazic-desktop
EOF

echo "Packaged to $OUT_DIR"
