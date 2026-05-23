param(
    [ValidateSet("go","llvm","all")] [string]$Backend = "all"
)

$scriptPath = $MyInvocation.MyCommand.Path
$rootDir = Split-Path -Parent $scriptPath
$rootDir = Split-Path -Parent $rootDir
$env:BAZIC_HOME = $rootDir
. (Join-Path $rootDir "scripts\bench_manifest.ps1")
$benchManifest = Get-BenchManifest $rootDir
$benchFiles = Get-BenchEntries $benchManifest

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
