#!/usr/bin/env bash
set -euo pipefail

OUT_DIR="${1:-dist/release}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

declare -a targets=(
  "windows/amd64/.exe"
  "linux/amd64/"
  "darwin/amd64/"
  "darwin/arm64/"
)

for t in "${targets[@]}"; do
  IFS="/" read -r os arch ext <<< "$t"
  dir="$OUT_DIR/${os}-${arch}"
  mkdir -p "$dir"

  GOOS="$os" GOARCH="$arch" go build -trimpath -ldflags "-buildid=" -o "$dir/bazic$ext" ./cmd/bazic
  GOOS="$os" GOARCH="$arch" go build -trimpath -ldflags "-buildid=" -o "$dir/bazc$ext" ./cmd/bazc
  GOOS="$os" GOARCH="$arch" go build -trimpath -ldflags "-buildid=" -o "$dir/bazlsp$ext" ./cmd/bazlsp

  rm -rf "$dir/std"
  cp -R ./std "$dir/std"
  rm -rf "$dir/runtime"
  cp -R ./runtime "$dir/runtime"
  if [ -f "./tools/vscode/baziclang-0.1.0.vsix" ]; then
    cp "./tools/vscode/baziclang-0.1.0.vsix" "$dir/baziclang.vsix"
  fi
done

echo "Release artifacts in $OUT_DIR"
