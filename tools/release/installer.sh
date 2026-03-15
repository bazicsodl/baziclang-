#!/usr/bin/env bash
set -euo pipefail

OUT_DIR="${1:-dist/release}"
PACKAGE_DIR="${2:-dist/packages}"
VERSION="${3:-1.0.0}"
IDENTIFIER="${BAZIC_MAC_PKG_ID:-com.baziclang.bazic}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

if ! command -v pkgbuild >/dev/null 2>&1; then
  echo "pkgbuild not found. Install Xcode command line tools to build PKG."
  exit 0
fi

mkdir -p "$PACKAGE_DIR"

for dir in "$OUT_DIR"/darwin-*; do
  [[ -d "$dir" ]] || continue
  name="$(basename "$dir")"
  pkg="$PACKAGE_DIR/bazic-$name.pkg"
  pkgbuild --root "$dir" --identifier "$IDENTIFIER.$name" --version "$VERSION" "$pkg"
  echo "Wrote $pkg"
done
