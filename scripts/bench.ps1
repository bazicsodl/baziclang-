param(
    [ValidateSet("go","llvm","all")] [string]$Backend = "all"
)

$scriptPath = $MyInvocation.MyCommand.Path
$rootDir = Split-Path -Parent $scriptPath
$rootDir = Split-Path -Parent $rootDir
$env:BAZIC_HOME = $rootDir

$benchFiles = @(
    @{ Name = "string_concat"; Path = (Join-Path "bench" "string_concat.bz") },
    @{ Name = "string_builder"; Path = (Join-Path "bench" "string_builder.bz") },
    @{ Name = "json_validate"; Path = (Join-Path "bench" "json_validate.bz") },
    @{ Name = "crypto_sha256"; Path = (Join-Path "bench" "crypto_sha256.bz") },
    @{ Name = "parse_int_float"; Path = (Join-Path "bench" "parse_int_float.bz") },
    @{ Name = "loop_arith"; Path = (Join-Path "bench" "loop_arith.bz") },
    @{ Name = "match_hot"; Path = (Join-Path "bench" "match_hot.bz") },
    @{ Name = "base64_roundtrip"; Path = (Join-Path "bench" "base64_roundtrip.bz") },
    @{ Name = "jwt_sign_verify"; Path = (Join-Path "bench" "jwt_sign_verify.bz") }
)

function Run-Bench($backend) {
    Write-Host "== Backend: $backend =="
    foreach ($b in $benchFiles) {
        $output = & go run ./cmd/bazc run $b.Path --backend $backend 2>$null
        if ($LASTEXITCODE -ne 0) {
            Write-Host ("{0,-18} {1}" -f $b.Name, "error")
            continue
        }
        $timeMs = $output | Where-Object { "$_" -match '^[0-9]+$' } | Select-Object -First 1
        if ($timeMs) {
            Write-Host ("{0,-18} {1,8} ms" -f $b.Name, $timeMs)
        } else {
            Write-Host ("{0,-18} {1}" -f $b.Name, "error")
        }
    }
    Write-Host ""
}

if ($Backend -eq "all") {
    Run-Bench "go"
    Run-Bench "llvm"
} else {
    Run-Bench $Backend
}
