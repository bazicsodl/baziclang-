param(
    [ValidateSet("windows","linux","macos")] [string]$Platform = "windows",
    [int]$Iterations = 3
)

$platformKey = $Platform.ToLowerInvariant()
$scriptPath = $MyInvocation.MyCommand.Path
$rootDir = Split-Path -Parent $scriptPath
$rootDir = Split-Path -Parent $rootDir
$env:BAZIC_HOME = $rootDir
. (Join-Path $rootDir "scripts\bench_manifest.ps1")
$benchManifest = Get-BenchManifest $rootDir
$benchFiles = Get-BenchEntries $benchManifest
$thresholdPercent = Get-BenchThresholdPercent $benchManifest

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
    throw ("failed to build bench capture compiler binary: {0}" -f $rendered)
}
& $bazcBin pkg sync 2>$null
if ($LASTEXITCODE -ne 0) {
    throw "failed to sync packages for bench capture"
}

function Run-Bench($backend, $name, $path, $iters) {
    $best = -1
    $lastFailure = $null
    for ($i = 0; $i -lt $iters; $i++) {
        $output = & $bazcBin run --backend $backend $path 2>&1
        $exitCode = $LASTEXITCODE
        $timeMs = $null
        if ($exitCode -eq 0) {
            $timeMs = $output | Where-Object { "$_" -match '^[0-9]+$' } | Select-Object -First 1
        }
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

$go = @{}
$llvm = @{}
foreach ($b in $benchFiles) {
    $go[$b.Name] = Run-Bench "go" $b.Name $b.Path $Iterations
    $llvm[$b.Name] = Run-Bench "llvm" $b.Name $b.Path $Iterations
}

$doc = New-Object System.Xml.XmlDocument
$root = $doc.CreateElement("benchmarks")
$baseline = $doc.CreateElement("baseline")
$baseline.SetAttribute("platform", $platformKey)
$baseline.SetAttribute("threshold_percent", [string]$thresholdPercent)

$goNode = $doc.CreateElement("go")
foreach ($k in $go.Keys) {
    $n = $doc.CreateElement($k)
    $n.InnerText = [string]$go[$k]
    $goNode.AppendChild($n) | Out-Null
}

$llvmNode = $doc.CreateElement("llvm")
foreach ($k in $llvm.Keys) {
    $n = $doc.CreateElement($k)
    $n.InnerText = [string]$llvm[$k]
    $llvmNode.AppendChild($n) | Out-Null
}

$baseline.AppendChild($goNode) | Out-Null
$baseline.AppendChild($llvmNode) | Out-Null
$root.AppendChild($baseline) | Out-Null
$doc.AppendChild($root) | Out-Null

$benchDir = "bench"
$compatPath = Join-Path $benchDir "baseline.xml"
$platformPath = Join-Path $benchDir ("baseline.{0}.xml" -f $platformKey)
$doc.Save($compatPath)
$doc.Save($platformPath)
Write-Host ("Saved {0}" -f $compatPath)
Write-Host "Saved $platformPath"
exit 0
