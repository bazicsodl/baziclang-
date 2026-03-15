$root = Split-Path -Parent $PSScriptRoot
$root = Split-Path -Parent $root
Set-Location $root

$bazic = Join-Path $root "bin\\bazic.exe"
if (-not (Test-Path $bazic)) {
    throw "bazic.exe not found. Build it with: go build .\\cmd\\bazic -o .\\bin\\bazic.exe"
}

& $bazic build .\examples\web\app.bz --target wasm --backend go -o .\examples\web\app.wasm
$goroot = & go env GOROOT
$candidate = Join-Path $goroot "misc\wasm\wasm_exec.js"
if (-not (Test-Path $candidate)) {
    $candidate = Get-ChildItem -Recurse -Filter wasm_exec.js $goroot | Select-Object -First 1 -ExpandProperty FullName
}
if (Test-Path $candidate) {
    Copy-Item $candidate .\examples\web\wasm_exec.js -Force
}

Write-Host "Built .\examples\web\app.wasm"
