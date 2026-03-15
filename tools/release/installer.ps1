param(
    [string]$Version = "1.0.0",
    [string]$OutDir = "dist\\release",
    [string]$PackageDir = "dist\\packages",
    [string]$UpgradeCode = ""
)

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$root = Split-Path -Parent $root
$root = Split-Path -Parent $root
Set-Location $root

$heat = (Get-Command "heat.exe" -ErrorAction SilentlyContinue).Source
$candle = (Get-Command "candle.exe" -ErrorAction SilentlyContinue).Source
$light = (Get-Command "light.exe" -ErrorAction SilentlyContinue).Source
if (-not $heat -or -not $candle -or -not $light) {
    $wixBin = Join-Path $root "tools\\release\\wixbin"
    if (Test-Path $wixBin) {
        $heat = Join-Path $wixBin "heat.exe"
        $candle = Join-Path $wixBin "candle.exe"
        $light = Join-Path $wixBin "light.exe"
        if (-not (Test-Path $heat) -or -not (Test-Path $candle) -or -not (Test-Path $light)) {
            $heat = $null
            $candle = $null
            $light = $null
        }
    }
}
if (-not $heat -or -not $candle -or -not $light) {
    Write-Host "WiX toolset not found (heat.exe/candle.exe/light.exe). Install WiX v3+ or run tools\\release\\install_wix.ps1."
    exit 0
}

$winDir = Join-Path $OutDir "windows-amd64"
if (-not (Test-Path $winDir)) {
    Write-Host "Missing $winDir. Run tools\\release\\build.ps1 first."
    exit 1
}

$vsixSource = Join-Path $root "tools\\vscode\\baziclang-0.1.0.vsix"
$vsixOut = Join-Path $winDir "baziclang.vsix"
if ((Test-Path $vsixSource) -and -not (Test-Path $vsixOut)) {
    Copy-Item -Force $vsixSource $vsixOut
}

if (-not $UpgradeCode) {
    $UpgradeCode = "B6AE2C34-2C01-4EE0-9B55-0C7E6E52AB10"
}

$wixDir = Join-Path $root "tools\\release\\wix"
$tmpDir = Join-Path $root "dist\\wix"
New-Item -ItemType Directory -Force -Path $tmpDir | Out-Null

$productTemplate = Get-Content -Raw (Join-Path $wixDir "product.wxs")
$productPath = Join-Path $tmpDir "product.wxs"
$productText = $productTemplate.Replace("{{Version}}", $Version).Replace("{{UpgradeCode}}", $UpgradeCode).Replace("{{SourceDir}}", $winDir)
Set-Content -Path $productPath -Value $productText -Encoding UTF8

$stdDir = Join-Path $winDir "std"
$stdWxs = Join-Path $tmpDir "std.wxs"
& $heat dir $stdDir -cg StdFiles -dr INSTALLFOLDER -gg -sreg -srd -sfrag -out $stdWxs
if ($LASTEXITCODE -ne 0) { throw "heat failed" }

$runtimeDir = Join-Path $winDir "runtime"
$runtimeWxs = Join-Path $tmpDir "runtime.wxs"
& $heat dir $runtimeDir -cg RuntimeFiles -dr INSTALLFOLDER -gg -sreg -srd -sfrag -out $runtimeWxs
if ($LASTEXITCODE -ne 0) { throw "heat failed (runtime)" }

$obj1 = Join-Path $tmpDir "product.wixobj"
$obj2 = Join-Path $tmpDir "std.wixobj"
$obj3 = Join-Path $tmpDir "runtime.wixobj"
& $candle -nologo -out $obj1 $productPath
if ($LASTEXITCODE -ne 0) { throw "candle failed (product)" }
& $candle -nologo -out $obj2 $stdWxs
if ($LASTEXITCODE -ne 0) { throw "candle failed (std)" }
& $candle -nologo -out $obj3 $runtimeWxs
if ($LASTEXITCODE -ne 0) { throw "candle failed (runtime)" }

New-Item -ItemType Directory -Force -Path $PackageDir | Out-Null
$msi = Join-Path $PackageDir "bazic-windows-amd64.msi"
& $light -nologo -ext WixUIExtension -out $msi $obj1 $obj2 $obj3 -b $winDir -b $stdDir -b $runtimeDir
if ($LASTEXITCODE -ne 0) { throw "light failed" }

Write-Host "Wrote $msi"
