param(
  [string]$OutDir = ".\\bin"
)

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

New-Item -ItemType Directory -Force $OutDir | Out-Null

$bazic = Join-Path $OutDir "bazic.exe"
$bazc = Join-Path $OutDir "bazc.exe"
$bazlsp = Join-Path $OutDir "bazlsp.exe"
$std = Join-Path $root "std"
$stdOut = Join-Path $OutDir "std"
$runtime = Join-Path $root "runtime"
$runtimeOut = Join-Path $OutDir "runtime"
$vsixSource = Join-Path $root "tools\\vscode\\baziclang-0.1.0.vsix"
$vsixOut = Join-Path $OutDir "baziclang.vsix"

go build -o $bazic .\cmd\bazic
if ($LASTEXITCODE -ne 0) { throw "build failed: bazic" }
go build -o $bazc .\cmd\bazc
if ($LASTEXITCODE -ne 0) { throw "build failed: bazc" }
go build -o $bazlsp .\cmd\bazlsp
if ($LASTEXITCODE -ne 0) { throw "build failed: bazlsp" }

if (Test-Path $std) {
  if (Test-Path $stdOut) {
    Remove-Item -Recurse -Force $stdOut
  }
  Copy-Item $std $stdOut -Recurse -Force
  Write-Host "Copied stdlib to $stdOut"
}
if (Test-Path $runtime) {
  if (Test-Path $runtimeOut) {
    Remove-Item -Recurse -Force $runtimeOut
  }
  Copy-Item $runtime $runtimeOut -Recurse -Force
  Write-Host "Copied runtime to $runtimeOut"
}

if (Test-Path $vsixSource) {
  Copy-Item $vsixSource $vsixOut -Force
  Write-Host "Copied VS Code extension to $vsixOut"
  $codeCmd = (Get-Command "code" -ErrorAction SilentlyContinue).Source
  if (-not $codeCmd) {
    $codeCmd = (Get-Command "code.cmd" -ErrorAction SilentlyContinue).Source
  }
  if ($codeCmd) {
    & $codeCmd --install-extension $vsixOut --force | Out-Null
    Write-Host "Installed Bazic VS Code extension"
  }
}

Write-Host "Built $bazic"
Write-Host "Built $bazc"
Write-Host "Built $bazlsp"
