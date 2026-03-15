#!/usr/bin/env bash
set -euo pipefail

OUT_DIR="${1:-dist/release}"
PACKAGE_DIR="${2:-dist/packages}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

if [[ ! -d "$OUT_DIR" ]]; then
  ./tools/release/build.sh "$OUT_DIR"
fi

mkdir -p "$PACKAGE_DIR"

echo "Generating SBOM..."
go run ./cmd/bazic pkg sbom -o "$OUT_DIR/bazic.sbom.json"

declare -a targets=(
  "windows-amd64:zip"
  "linux-amd64:tar"
  "darwin-amd64:tar"
  "darwin-arm64:tar"
)

for t in "${targets[@]}"; do
  IFS=":" read -r name kind <<< "$t"
  dir="$OUT_DIR/$name"
  [[ -d "$dir" ]] || continue
  base="$PACKAGE_DIR/bazic-$name"
  if [[ "$kind" == "zip" ]]; then
    zipfile="$base.zip"
    rm -f "$zipfile"
    (cd "$dir" && zip -r -q "$zipfile" .)
    echo "Wrote $zipfile"
  else
    tarfile="$base.tar.gz"
    rm -f "$tarfile"
    tar -czf "$tarfile" -C "$OUT_DIR" "$name"
    echo "Wrote $tarfile"
  fi
done

checksum="$PACKAGE_DIR/SHA256SUMS.txt"
rm -f "$checksum"
for f in "$PACKAGE_DIR"/*.zip "$PACKAGE_DIR"/*.tar.gz "$OUT_DIR/bazic.sbom.json"; do
  [[ -f "$f" ]] || continue
  hash="$(sha256sum "$f" | awk '{print $1}')"
  rel="${f#$ROOT_DIR/}"
  echo "$hash  $rel" >> "$checksum"
done

echo "Checksums written to $checksum"
