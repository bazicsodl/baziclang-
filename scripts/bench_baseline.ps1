param(
    [ValidateSet("windows","linux","macos")] [string]$Platform = "windows",
    [int]$Iterations = 3
)

$platformKey = $Platform.ToLowerInvariant()
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

$bazcName = if ($IsWindows) { "bazc-bench.exe" } else { "bazc-bench" }
$bazcBin = Join-Path ([System.IO.Path]::GetTempPath()) $bazcName
& go build -buildvcs=false -o $bazcBin ./cmd/bazc 2>$null
if ($LASTEXITCODE -ne 0 -or -not (Test-Path $bazcBin)) {
    throw "failed to build bench capture compiler binary"
}
& $bazcBin pkg sync 2>$null
if ($LASTEXITCODE -ne 0) {
    throw "failed to sync packages for bench capture"
}

function Run-Bench($backend, $name, $path, $iters) {
    $best = -1
    $lastFailure = $null
    for ($i = 0; $i -lt $iters; $i++) {
        $output = & $bazcBin run $path --backend $backend 2>&1
        $timeMs = $output | Select-Object -First 1
        if ($timeMs -match '^[0-9]+$') {
            $val = [int]$timeMs
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
$baseline.SetAttribute("threshold_percent", "40")

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
