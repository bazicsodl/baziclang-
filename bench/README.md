# Benchmarks

This folder contains small, focused performance tests. Run with:

```powershell
.\bin\bazic.exe run .\bench\<name>.bz
```

## Included
- `loop_arith`
- `string_concat`
- `string_builder`
- `parse_int_float`
- `json_validate`
- `crypto_sha256`
- `base64_roundtrip`
- `match_hot`
- `jwt_sign_verify`

## Baseline Capture

Capture platform baselines with:

```powershell
./scripts/bench_baseline.ps1 -Platform windows
./scripts/bench_baseline.ps1 -Platform linux
./scripts/bench_baseline.ps1 -Platform macos
```

This writes:
- `bench/baseline.xml`
- `bench/baseline.<platform>.xml`

`bench/baseline.<platform>.xml` is the authoritative per-platform baseline file.
`bench/baseline.xml` remains the latest captured baseline snapshot for compatibility.

The CI `bench-baseline-capture` job uploads those files as per-platform artifacts when the workflow is run manually.

To promote a downloaded artifact into the repo:

```powershell
./scripts/bench_promote.ps1 -Platform windows -Source .\bench-baseline-Windows\baseline.windows.xml -UpdateCompatibilitySnapshot
./scripts/bench_promote.ps1 -Platform linux -Source .\bench-baseline-Linux\baseline.linux.xml
./scripts/bench_promote.ps1 -Platform macos -Source .\bench-baseline-macOS\baseline.macos.xml
```
