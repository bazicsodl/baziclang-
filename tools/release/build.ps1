param(
    [string]$OutDir = "dist\\release"
)

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$root = Split-Path -Parent $root
$root = Split-Path -Parent $root
Set-Location $root

$targets = @(
    @{ Os = "windows"; Arch = "amd64"; Ext = ".exe" },
    @{ Os = "linux"; Arch = "amd64"; Ext = "" },
    @{ Os = "darwin"; Arch = "amd64"; Ext = "" },
    @{ Os = "darwin"; Arch = "arm64"; Ext = "" }
)

foreach ($t in $targets) {
    $dir = Join-Path $OutDir ("{0}-{1}" -f $t.Os, $t.Arch)
    New-Item -ItemType Directory -Force -Path $dir | Out-Null
    $env:GOOS = $t.Os
    $env:GOARCH = $t.Arch

    $bazic = Join-Path $dir ("bazic" + $t.Ext)
    $bazc = Join-Path $dir ("bazc" + $t.Ext)
    $bazlsp = Join-Path $dir ("bazlsp" + $t.Ext)
    go build -trimpath -ldflags "-buildid=" -o $bazic .\cmd\bazic
    if ($LASTEXITCODE -ne 0) { throw "build failed: bazic ($($t.Os)/$($t.Arch))" }
    go build -trimpath -ldflags "-buildid=" -o $bazc .\cmd\bazc
    if ($LASTEXITCODE -ne 0) { throw "build failed: bazc ($($t.Os)/$($t.Arch))" }
    go build -trimpath -ldflags "-buildid=" -o $bazlsp .\cmd\bazlsp
    if ($LASTEXITCODE -ne 0) { throw "build failed: bazlsp ($($t.Os)/$($t.Arch))" }

    Copy-Item -Recurse -Force .\std (Join-Path $dir "std")
    Copy-Item -Recurse -Force .\runtime (Join-Path $dir "runtime")
    $vsixSource = Join-Path $root "tools\\vscode\\baziclang-0.1.0.vsix"
    if (Test-Path $vsixSource) {
        Copy-Item -Force $vsixSource (Join-Path $dir "baziclang.vsix")
    }
}

Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue

Write-Host "Release artifacts in $OutDir"
