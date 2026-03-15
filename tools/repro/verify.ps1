param(
  [string]$Input = ".\examples\phase3\main.bz",
  [string]$OutDir = ".\out_repro"
)

New-Item -ItemType Directory -Force $OutDir | Out-Null
$one = Join-Path $OutDir "app1.exe"
$two = Join-Path $OutDir "app2.exe"
$bazic = Join-Path (Split-Path -Parent (Split-Path -Parent $PSScriptRoot)) "bin\\bazic.exe"
if (-not (Test-Path $bazic)) {
  throw "bazic.exe not found. Build it with: go build .\\cmd\\bazic -o .\\bin\\bazic.exe"
}

& $bazic build $Input --backend go -o $one
Start-Sleep -Milliseconds 500

& $bazic build $Input --backend go -o $two

$h1 = (Get-FileHash $one -Algorithm SHA256).Hash
$h2 = (Get-FileHash $two -Algorithm SHA256).Hash

Write-Host "app1:" $h1
Write-Host "app2:" $h2

if ($h1 -ne $h2) {
  Write-Error "reproducibility check failed"
  exit 1
}

Write-Host "reproducibility check passed"
