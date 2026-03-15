param(
    [string]$OutDir = "dist\\release"
)

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$root = Split-Path -Parent $root
$root = Split-Path -Parent $root
Set-Location $root

$pfx = $env:BAZIC_WIN_CERT_PFX
$pfxPass = $env:BAZIC_WIN_CERT_PASS
$timestamp = $env:BAZIC_WIN_TIMESTAMP_URL
if (-not $timestamp) { $timestamp = "http://timestamp.digicert.com" }

if (-not $pfx -or -not (Test-Path $pfx)) {
    Write-Host "Skipping Windows signing: set BAZIC_WIN_CERT_PFX to a valid .pfx path."
    exit 0
}

if (-not $pfxPass) {
    Write-Host "Skipping Windows signing: set BAZIC_WIN_CERT_PASS."
    exit 0
}

$signtool = "signtool.exe"
$signtoolPath = (Get-Command $signtool -ErrorAction SilentlyContinue).Source
if (-not $signtoolPath) {
    Write-Host "Skipping Windows signing: signtool.exe not found."
    exit 0
}

$winDir = Join-Path $OutDir "windows-amd64"
if (-not (Test-Path $winDir)) {
    Write-Host "No windows-amd64 release directory found."
    exit 0
}

$files = @()
$files += Get-ChildItem -Path $winDir -Filter "*.exe" -File

$msi = Join-Path $root "dist\\packages\\bazic-windows-amd64.msi"
if (Test-Path $msi) {
    $files += Get-Item $msi
}

if ($files.Count -eq 0) {
    Write-Host "No Windows binaries to sign."
    exit 0
}

foreach ($f in $files) {
    & $signtoolPath sign /f $pfx /p $pfxPass /tr $timestamp /td sha256 /fd sha256 $f.FullName
    if ($LASTEXITCODE -ne 0) { throw "signtool failed for $($f.FullName)" }
    Write-Host "Signed $($f.Name)"
}

Write-Host "Windows signing complete."
