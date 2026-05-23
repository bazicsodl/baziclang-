param(
    [ValidateSet("go","llvm")] [string]$Backend,
    [int]$ThresholdPercent,
    [int]$Iterations = 3,
    [ValidateSet("auto","baseline","ratio")] [string]$Mode = "auto",
    [ValidateSet("auto","windows","linux","macos")] [string]$Platform = "auto"
)

if (-not $Backend) {
    Write-Host "Usage: scripts\\bench_gate.ps1 -Backend go|llvm [-ThresholdPercent <percent>]"
    exit 2
}

$scriptPath = $MyInvocation.MyCommand.Path
$rootDir = Split-Path -Parent $scriptPath
$rootDir = Split-Path -Parent $rootDir
$env:BAZIC_HOME = $rootDir
$benchDir = Join-Path $rootDir "bench"
. (Join-Path $rootDir "scripts\bench_manifest.ps1")
$benchManifest = Get-BenchManifest $rootDir
$benchFiles = Get-BenchEntries $benchManifest
$targets = @{}
foreach ($b in $benchFiles) {
    $targets[$b.Name] = [double]$b.LLVMRatioTarget
}
if (-not $PSBoundParameters.ContainsKey("ThresholdPercent")) {
    $ThresholdPercent = Get-BenchThresholdPercent $benchManifest
}

$isWindowsHost = $false
if ($null -ne $IsWindows) {
    $isWindowsHost = [bool]$IsWindows
} elseif ($env:OS) {
    $isWindowsHost = $env:OS.ToLowerInvariant().Contains("windows")
}
$bazcName = if ($isWindowsHost) { "bazc-bench.exe" } else { "bazc-bench" }
$bazcBin = Join-Path ([System.IO.Path]::GetTempPath()) $bazcName
$buildOutput = & go build -buildvcs=false -o $bazcBin ./cmd/bazc 2>&1
if ($LASTEXITCODE -ne 0 -or -not (Test-Path $bazcBin)) {
    $rendered = (($buildOutput | ForEach-Object { "$_" }) -join [Environment]::NewLine).Trim()
    if (-not $rendered) {
        $rendered = "no output"
    }
    throw ("failed to build bench gate compiler binary: {0}" -f $rendered)
}
& $bazcBin pkg sync 2>$null
if ($LASTEXITCODE -ne 0) {
    throw "failed to sync packages for bench gate"
}

function Run-Bench($backend, $name, $path, $iters) {
    $best = -1
    $lastFailure = $null
    for ($i = 0; $i -lt $iters; $i++) {
        Write-Host ("-> {0} [{1}] iteration {2}/{3}" -f $name, $backend, ($i + 1), $iters)
        $output = & $bazcBin run --backend $backend $path 2>&1
        $exitCode = $LASTEXITCODE
        if ($exitCode -ne 0) {
            $rendered = (($output | ForEach-Object { "$_" }) -join [Environment]::NewLine).Trim()
            if (-not $rendered) {
                $rendered = "no output"
            }
            $lastFailure = $rendered
            continue
        }
        $timeMs = $output | Where-Object { "$_" -match '^[0-9]+$' } | Select-Object -First 1
        if ($timeMs) {
            $val = [int]"$timeMs"
            if ($best -lt 0 -or $val -lt $best) { $best = $val }
            continue
        }
        $rendered = (($output | ForEach-Object { "$_" }) -join [Environment]::NewLine).Trim()
        if (-not $rendered) {
            $rendered = "no output"
        }
        $lastFailure = $rendered
    }
    if ($best -lt 0) {
        throw ("benchmark '{0}' failed for backend '{1}': {2}" -f $name, $backend, $lastFailure)
    }
    return $best
}

function New-EmptyBaselineMap($entries) {
    $map = @{}
    foreach ($entry in $entries) {
        $map[$entry.Name] = 0
    }
    return $map
}

$baseline = @{}
$baseline["go"] = New-EmptyBaselineMap $benchFiles
$baseline["llvm"] = New-EmptyBaselineMap $benchFiles
$baseline["go"]["string_concat"] = 119
$baseline["go"]["string_builder"] = 8935
$baseline["go"]["json_validate"] = 11
$baseline["go"]["crypto_sha256"] = 14
$baseline["go"]["parse_int_float"] = 21
$baseline["llvm"]["string_concat"] = 182
$baseline["llvm"]["string_builder"] = 27956
$baseline["llvm"]["json_validate"] = 5
$baseline["llvm"]["crypto_sha256"] = 15
$baseline["llvm"]["parse_int_float"] = 34

$baselinePlatformKey = $Platform
$os = $env:OS
if ($baselinePlatformKey -eq "auto") {
    if ($os -and $os.ToLower() -like "*windows*") {
        $baselinePlatformKey = "windows"
    } elseif ($IsLinux) {
        $baselinePlatformKey = "linux"
    } elseif ($IsMacOS) {
        $baselinePlatformKey = "macos"
    } else {
        $baselinePlatformKey = "unknown"
    }
}

$baselineCandidates = @()
if ($baselinePlatformKey -ne "unknown") {
    $baselineCandidates += (Join-Path $benchDir ("baseline.{0}.xml" -f $baselinePlatformKey))
}
$baselineCandidates += (Join-Path $benchDir "baseline.xml")
$baselinePlatform = $null
foreach ($baselineXml in $baselineCandidates) {
    if (-not (Test-Path $baselineXml)) {
        continue
    }
    try {
        [xml]$doc = Get-Content -Raw $baselineXml -ErrorAction Stop
    } catch {
        $doc = $null
    }
    if ($doc -ne $null) {
        $baseNode = $doc.benchmarks.baseline
        if ($baseNode -and $baseNode.threshold_percent -and -not $PSBoundParameters.ContainsKey("ThresholdPercent")) {
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
        break
    }
}

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
        try {
            $tGo = Run-Bench "go" $b.Name $b.Path $Iterations
        } catch {
            Write-Host ("{0,-18} go error: {1}" -f $b.Name, $_.Exception.Message)
            $fail = $true
            continue
        }
        try {
            $tLlvm = Run-Bench "llvm" $b.Name $b.Path $Iterations
        } catch {
            Write-Host ("{0,-18} llvm error: {1}" -f $b.Name, $_.Exception.Message)
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
