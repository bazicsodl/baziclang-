# Performance Benchmarks

This repo tracks two benchmark classes:
- **Compiler benchmarks** (Go tests).
- **Runtime benchmarks** (Bazic programs, run via `bazc`).

## Compiler Benchmarks
```powershell
go test ./internal/compiler -run ^$ -bench . -benchmem
```

## Runtime Benchmarks
```powershell
.\scripts\bench.ps1
```

To gate regressions (default 3 iterations):
```powershell
.\scripts\bench_gate.ps1 -Backend go
.\scripts\bench_gate.ps1 -Backend llvm
```

Gate runs each benchmark multiple times and uses the fastest time to reduce noise.
The gate prefers `bench/baseline.<platform>.xml` for the current platform and falls back to `bench/baseline.xml`.
If no matching platform baseline exists, non-Windows platforms automatically switch to **ratio mode** (LLVM vs Go).

To record a fresh baseline:
```powershell
.\scripts\bench_baseline.ps1 -Platform windows
```

This writes:
- `bench/baseline.xml`
- `bench/baseline.windows.xml`

Linux/macOS can run the same script with `-Platform linux` or `-Platform macos`
to produce `bench/baseline.linux.xml` and `bench/baseline.macos.xml`.

CI can capture baselines via the `bench-baseline-capture` workflow (manual dispatch).
Promote a downloaded artifact with:
```powershell
.\scripts\bench_promote.ps1 -Platform windows -Source .\bench-baseline-Windows\baseline.windows.xml -UpdateCompatibilitySnapshot
.\scripts\bench_promote.ps1 -Platform linux -Source .\bench-baseline-Linux\baseline.linux.xml
.\scripts\bench_promote.ps1 -Platform macos -Source .\bench-baseline-macOS\baseline.macos.xml
```

Audit the committed baseline set with:
```powershell
.\scripts\bench_audit.ps1
.\scripts\bench_audit.ps1 -RequireAll
```

The CI `bench-baseline-audit` job runs the non-strict audit on every change.
For release-readiness checks, dispatch the workflow manually with `release_readiness=true`.
That turns on the strict audit automatically and requires committed Windows, Linux, and macOS baseline files.

Bench programs live in `bench/`:
- `bench/string_concat.bz`
- `bench/string_builder.bz`
- `bench/json_validate.bz`
- `bench/crypto_sha256.bz`
- `bench/parse_int_float.bz`
- `bench/loop_arith.bz`
- `bench/match_hot.bz`
- `bench/base64_roundtrip.bz`
- `bench/jwt_sign_verify.bz`

## Targets (v1.0)
These targets are for LLVM backend vs Go backend on the same machine:
- `string_concat`: LLVM <= 3.0x Go
- `string_builder`: LLVM <= 6.0x Go
- `json_validate`: LLVM <= 2.0x Go
- `crypto_sha256`: LLVM <= 1.5x Go
- `parse_int_float`: LLVM <= 3.0x Go
- `loop_arith`: LLVM <= 2.0x Go
- `match_hot`: LLVM <= 2.0x Go
- `base64_roundtrip`: LLVM <= 4.0x Go
- `jwt_sign_verify`: LLVM <= 2.0x Go

## Baseline (2026-03-06, Windows, local machine)
Go backend:
- `string_concat`: 119 ms
- `string_builder`: 8935 ms
- `json_validate`: 11 ms
- `crypto_sha256`: 14 ms
- `parse_int_float`: 21 ms
- `jwt_sign_verify`: 76 ms

LLVM backend:
- `string_concat`: 182 ms
- `string_builder`: 27956 ms
- `json_validate`: 5 ms
- `crypto_sha256`: 15 ms
- `parse_int_float`: 34 ms
- `jwt_sign_verify`: 107 ms

Baseline data is stored in:
- `bench/baseline.windows.xml` as the Windows platform baseline
- `bench/baseline.xml` as the latest compatibility snapshot for gating fallback

## Notes
- Benchmarks are deterministic and do not touch the network.
- Track results over time and gate regressions once stable baselines are established.
