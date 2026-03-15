param(
    [string]$OutDir = "dist\\desktop"
)

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$root = Split-Path -Parent $root
Set-Location $root

$src = "examples\\apps\\desktop\\main.bz"
$binDir = Join-Path $OutDir "bin"
New-Item -ItemType Directory -Force -Path $binDir | Out-Null

$exe = Join-Path $binDir "bazic-desktop.exe"
& go run .\cmd\bazc\ build $src -o $exe | Out-Null
if ($LASTEXITCODE -ne 0) {
    Write-Error "build failed"
    exit 1
}

$readme = Join-Path $OutDir "README.txt"
@"
Bazic Desktop MVP

Run:
  .\bin\bazic-desktop.exe
"@ | Set-Content $readme

Write-Host "Packaged to $OutDir"
