#!/usr/bin/env bash
set -euo pipefail

OUT_DIR="${1:-dist/release}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

IDENTITY="${BAZIC_MAC_IDENTITY:-}"
TEAM_ID="${BAZIC_MAC_TEAM_ID:-}"
APPLE_ID="${BAZIC_APPLE_ID:-}"
APPLE_PASS="${BAZIC_APPLE_PASS:-}"

if [[ -z "$IDENTITY" ]]; then
  echo "Skipping macOS signing: set BAZIC_MAC_IDENTITY."
  exit 0
fi

for dir in "$OUT_DIR"/darwin-*; do
  [[ -d "$dir" ]] || continue
  for bin in "$dir"/bazic "$dir"/bazc; do
    [[ -f "$bin" ]] || continue
    codesign --force --options runtime --timestamp --sign "$IDENTITY" "$bin"
    echo "Signed $bin"
  done
done

if [[ -n "$APPLE_ID" && -n "$APPLE_PASS" && -n "$TEAM_ID" ]]; then
  for dir in "$OUT_DIR"/darwin-*; do
    [[ -d "$dir" ]] || continue
    pkg="$dir/bazic-macos-binaries.zip"
    rm -f "$pkg"
    (cd "$dir" && zip -q -r "$pkg" bazic bazc std)
    xcrun notarytool submit "$pkg" --apple-id "$APPLE_ID" --password "$APPLE_PASS" --team-id "$TEAM_ID" --wait
    echo "Notarized $pkg"
  done
else
  echo "Skipping notarization: set BAZIC_APPLE_ID, BAZIC_APPLE_PASS, and BAZIC_MAC_TEAM_ID."
fi
