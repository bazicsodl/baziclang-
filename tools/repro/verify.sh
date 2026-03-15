#!/usr/bin/env bash
set -euo pipefail
INPUT=${1:-./examples/phase3/main.bz}
OUTDIR=${2:-./out_repro}
mkdir -p "$OUTDIR"
ONE="$OUTDIR/app1"
TWO="$OUTDIR/app2"

go run ./cmd/bazc build "$INPUT" -o "$ONE"
sleep 0.5
go run ./cmd/bazc build "$INPUT" -o "$TWO"

if command -v sha256sum >/dev/null; then
  H1=$(sha256sum "$ONE" | awk '{print $1}')
  H2=$(sha256sum "$TWO" | awk '{print $1}')
else
  H1=$(shasum -a 256 "$ONE" | awk '{print $1}')
  H2=$(shasum -a 256 "$TWO" | awk '{print $1}')
fi

echo "app1: $H1"
echo "app2: $H2"

if [ "$H1" != "$H2" ]; then
  echo "reproducibility check failed" >&2
  exit 1
fi

echo "reproducibility check passed"
