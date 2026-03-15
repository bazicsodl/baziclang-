param(
    [ValidateSet("go","llvm")] [string]$Backend,
    [int]$ThresholdPercent = 20,
    [int]$Iterations = 3,
    [ValidateSet("auto","baseline","ratio")] [string]$Mode = "auto"
)

if (-not $Backend) {
    Write-Host "Usage: scripts\\bench_gate.ps1 -Backend go|llvm [-ThresholdPercent 20]"
    exit 2
}

function Run-Bench($backend, $name, $path, $iters) {
    $best = -1
    for ($i = 0; $i -lt $iters; $i++) {
        $output = & go run .\cmd\bazc\ run $path --backend $backend 2>$null
        $timeMs = $output | Select-Object -First 1
        if ($timeMs -match '^[0-9]+$') {
            $val = [int]$timeMs
            if ($best -lt 0 -or $val -lt $best) { $best = $val }
        }
    }
    return $best
}

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

$baseline = @{}
$baseline["go"] = @{
    "string_concat" = 119
    "string_builder" = 8935
    "json_validate" = 11
    "crypto_sha256" = 14
    "parse_int_float" = 21
    "loop_arith" = 0
    "match_hot" = 0
    "base64_roundtrip" = 0
    "jwt_sign_verify" = 0
}
$baseline["llvm"] = @{
    "string_concat" = 182
    "string_builder" = 27956
    "json_validate" = 5
    "crypto_sha256" = 15
    "parse_int_float" = 34
    "loop_arith" = 0
    "match_hot" = 0
    "base64_roundtrip" = 0
    "jwt_sign_verify" = 0
}

$targets = @{
    "string_concat" = 3.0
    "string_builder" = 6.0
    "json_validate" = 2.0
    "crypto_sha256" = 1.5
    "parse_int_float" = 3.0
    "loop_arith" = 2.0
    "match_hot" = 2.0
    "base64_roundtrip" = 2.0
    "jwt_sign_verify" = 2.0
}

$scriptPath = $MyInvocation.MyCommand.Path
$rootDir = Split-Path -Parent $scriptPath
$rootDir = Split-Path -Parent $rootDir
$baselineXml = Join-Path $rootDir "bench\\baseline.xml"
$baselinePlatform = $null
if (Test-Path $baselineXml) {
    try {
        [xml]$doc = Get-Content -Raw $baselineXml -ErrorAction Stop
    } catch {
        $doc = $null
    }
    if ($doc -ne $null) {
        $baseNode = $doc.benchmarks.baseline
        if ($baseNode -and $baseNode.threshold_percent) {
            $ThresholdPercent = [int]$baseNode.threshold_percent
        }
        if ($baseNode -and $baseNode.platform) {
            $baselinePlatform = [string]$baseNode.platform
        }
        $goNode = $baseNode.go
        $llvmNode = $baseNode.llvm
        $keys = @($baseline["go"].Keys)
        foreach ($k in $keys) {
            if ($goNode.$k) { $baseline["go"][$k] = [int]$goNode.$k }
            if ($llvmNode.$k) { $baseline["llvm"][$k] = [int]$llvmNode.$k }
        }
    }
}

$os = $env:OS
if (-not $os) { $os = "unknown" }
$os = $os.ToLower()
if ($Mode -eq "auto") {
    if ($Backend -eq "llvm") {
        $Mode = "ratio"
    } elseif ($baselinePlatform -and $baselinePlatform.ToLower() -eq "windows" -and $os -like "*windows*") {
        $Mode = "baseline"
    } else {
        $Mode = "ratio"
    }
}

$fail = $false
if ($Mode -eq "baseline") {
    Write-Host "== Gate (baseline): $Backend (threshold ${ThresholdPercent}%, iterations ${Iterations}) =="
    foreach ($b in $benchFiles) {
        $t = Run-Bench $Backend $b.Name $b.Path $Iterations
        if ($t -lt 0) {
            Write-Host ("{0,-18} {1}" -f $b.Name, "error")
            $fail = $true
            continue
        }
        $base = $baseline[$Backend][$b.Name]
        if (-not $base -or $base -le 0) {
            Write-Host ("{0,-18} {1}" -f $b.Name, "skip (no baseline)")
            continue
        }
        $limit = [int][math]::Ceiling($base * (1 + $ThresholdPercent / 100.0))
        $status = if ($t -le $limit) { "OK" } else { "REGRESSION" }
        Write-Host ("{0,-18} {1,8} ms  (baseline {2} ms, limit {3} ms)  {4}" -f $b.Name, $t, $base, $limit, $status)
        if ($t -gt $limit) { $fail = $true }
    }
} else {
    Write-Host "== Gate (ratio): llvm vs go (iterations ${Iterations}) =="
    foreach ($b in $benchFiles) {
        $tGo = Run-Bench "go" $b.Name $b.Path $Iterations
        $tLlvm = Run-Bench "llvm" $b.Name $b.Path $Iterations
    if ($tGo -lt 0 -or $tLlvm -lt 0) {
        Write-Host ("{0,-18} {1}" -f $b.Name, "error")
        $fail = $true
        continue
    }
        $ratio = [math]::Round($tLlvm / $tGo, 2)
        $limit = $targets[$b.Name]
        if (-not $limit) {
            Write-Host ("{0,-18} {1}" -f $b.Name, "skip (no target)")
            continue
        }
        $status = if ($ratio -le $limit) { "OK" } else { "REGRESSION" }
        Write-Host ("{0,-18} llvm {1,6} ms / go {2,6} ms  ratio {3,4}x (limit {4}x)  {5}" -f $b.Name, $tLlvm, $tGo, $ratio, $limit, $status)
        if ($ratio -gt $limit) { $fail = $true }
    }
}

if ($fail) { exit 1 } else { exit 0 }
