param(
    [string]$OutDir = "dist\\release",
    [string]$PackageDir = "dist\\packages"
)

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$root = Split-Path -Parent $root
$root = Split-Path -Parent $root
Set-Location $root

if (-not (Test-Path $OutDir)) {
    & .\tools\release\build.ps1 -OutDir $OutDir
    if ($LASTEXITCODE -ne 0) { throw "build failed" }
}

New-Item -ItemType Directory -Force -Path $PackageDir | Out-Null

Write-Host "Generating SBOM..."
& go run .\cmd\bazic\ pkg sbom -o (Join-Path $OutDir "bazic.sbom.json")
if ($LASTEXITCODE -ne 0) { throw "sbom failed" }

$targets = @(
    @{ Name = "windows-amd64"; Kind = "zip" },
    @{ Name = "linux-amd64"; Kind = "tar" },
    @{ Name = "darwin-amd64"; Kind = "tar" },
    @{ Name = "darwin-arm64"; Kind = "tar" }
)

foreach ($t in $targets) {
    $dir = Join-Path $OutDir $t.Name
    if (-not (Test-Path $dir)) { continue }
    $pkgBase = Join-Path $PackageDir ("bazic-{0}" -f $t.Name)
    if ($t.Kind -eq "zip") {
        $zip = $pkgBase + ".zip"
        if (Test-Path $zip) { Remove-Item -Force $zip }
        Compress-Archive -Path (Join-Path $dir "*") -DestinationPath $zip
        Write-Host "Wrote $zip"
    } else {
        $tar = $pkgBase + ".tar.gz"
        if (Test-Path $tar) { Remove-Item -Force $tar }
        & tar -czf $tar -C $OutDir $t.Name
        Write-Host "Wrote $tar"
    }
}

$checksumPath = Join-Path $PackageDir "SHA256SUMS.txt"
if (Test-Path $checksumPath) { Remove-Item -Force $checksumPath }
$files = @()
$files += Get-ChildItem -File -Path $PackageDir | Where-Object { $_.Name -match '\.(zip|tar\.gz|msi)$' }
$sbom = Join-Path $OutDir "bazic.sbom.json"
if (Test-Path $sbom) {
    $files += Get-Item $sbom
}
foreach ($f in $files) {
    $hash = (Get-FileHash -Algorithm SHA256 $f.FullName).Hash.ToLower()
    $rel = $f.FullName.Substring($root.Length + 1).Replace("\\", "/")
    Add-Content -Path $checksumPath -Value ("{0}  {1}" -f $hash, $rel)
}

Write-Host "Checksums written to $checksumPath"

$verifyPath = Join-Path $PackageDir "VERIFY_CHECKSUMS_WINDOWS.txt"
$verifyText = @'
Windows verification:
1) Get-FileHash -Algorithm SHA256 .\dist\packages\bazic-windows-amd64.msi
2) Compare with the matching line in dist\packages\SHA256SUMS.txt
'@
Set-Content -Path $verifyPath -Value $verifyText -Encoding UTF8
Write-Host "Wrote $verifyPath"
