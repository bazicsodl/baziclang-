param(
    [string]$Version = "3.11.2"
)

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$root = Split-Path -Parent $root
$root = Split-Path -Parent $root
Set-Location $root

$wixDir = Join-Path $root "tools\\release\\wixbin"
New-Item -ItemType Directory -Force -Path $wixDir | Out-Null

$zipName = "wix311-binaries.zip"
$url = "https://github.com/wixtoolset/wix3/releases/download/wix3112rtm/$zipName"
if ($env:BAZIC_WIX_URL) {
    $url = $env:BAZIC_WIX_URL
}
$zipPath = Join-Path $wixDir $zipName

Write-Host "Downloading WiX $Version..."
try {
    if (Get-Command "Start-BitsTransfer" -ErrorAction SilentlyContinue) {
        Start-BitsTransfer -Source $url -Destination $zipPath -ErrorAction Stop
    } else {
        Invoke-WebRequest -Uri $url -OutFile $zipPath -ErrorAction Stop
    }
} catch {
    Write-Host "Download failed. Check network access and DNS."
    exit 1
}

Write-Host "Extracting..."
try {
    Expand-Archive -Path $zipPath -DestinationPath $wixDir -Force -ErrorAction Stop
} catch {
    Write-Host "Extraction failed."
    exit 1
}

Write-Host "WiX installed to $wixDir"
