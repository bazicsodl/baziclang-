param(
    [ValidateSet("go","llvm","all")] [string]$Backend = "all"
)

$benchFiles = @(
    @{ Name = "string_concat"; Path = "bench\\string_concat.bz" },
    @{ Name = "string_builder"; Path = "bench\\string_builder.bz" },
    @{ Name = "json_validate"; Path = "bench\\json_validate.bz" },
    @{ Name = "crypto_sha256"; Path = "bench\\crypto_sha256.bz" },
    @{ Name = "parse_int_float"; Path = "bench\\parse_int_float.bz" },
    @{ Name = "loop_arith"; Path = "bench\\loop_arith.bz" },
    @{ Name = "match_hot"; Path = "bench\\match_hot.bz" },
    @{ Name = "base64_roundtrip"; Path = "bench\\base64_roundtrip.bz" },
    @{ Name = "jwt_sign_verify"; Path = "bench\\jwt_sign_verify.bz" }
)

function Run-Bench($backend) {
    Write-Host "== Backend: $backend =="
    foreach ($b in $benchFiles) {
        $output = & go run .\cmd\bazc\ run $b.Path --backend $backend 2>$null
        $timeMs = $output | Select-Object -First 1
        if ($timeMs -match '^[0-9]+$') {
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
