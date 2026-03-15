#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "$ROOT_DIR"

BAZIC="$ROOT_DIR/bin/bazic"
if [ ! -f "$BAZIC" ]; then
  echo "bazic not found. Build it with: go build ./cmd/bazic -o ./bin/bazic" >&2
  exit 1
fi

"$BAZIC" build ./examples/web/app.bz --target wasm --backend go -o ./examples/web/app.wasm
if ! command -v go >/dev/null; then
  echo "go not found (required to copy wasm_exec.js)" >&2
  exit 1
fi
GOROOT=$(go env GOROOT)
WASM_EXEC="$GOROOT/misc/wasm/wasm_exec.js"
if [ -f "$WASM_EXEC" ]; then
  cp "$WASM_EXEC" ./examples/web/wasm_exec.js
fi

echo "Built ./examples/web/app.wasm"
