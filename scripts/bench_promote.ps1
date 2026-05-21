param(
    [Parameter(Mandatory = $true)]
    [ValidateSet("windows","linux","macos")]
    [string]$Platform,

    [string]$Source = "",

    [switch]$UpdateCompatibilitySnapshot
)

$scriptPath = $MyInvocation.MyCommand.Path
$rootDir = Split-Path -Parent $scriptPath
$rootDir = Split-Path -Parent $rootDir

if (-not $Source) {
    $artifactDir = Join-Path $rootDir ("bench-baseline-{0}" -f $Platform)
    $candidate = Join-Path $artifactDir ("baseline.{0}.xml" -f $Platform)
    if (Test-Path $candidate) {
        $Source = $candidate
    } else {
        $Source = Join-Path $rootDir "bench\baseline.xml"
    }
}

if (-not (Test-Path $Source)) {
    Write-Error ("Baseline source not found: {0}" -f $Source)
    exit 1
}

try {
    [xml]$doc = Get-Content -Raw $Source -ErrorAction Stop
} catch {
    Write-Error ("Failed to parse baseline XML: {0}" -f $Source)
    exit 1
}

$baseline = $doc.benchmarks.baseline
if (-not $baseline) {
    Write-Error ("Baseline XML missing <baseline> node: {0}" -f $Source)
    exit 1
}

$baseline.SetAttribute("platform", $Platform)

$platformPath = Join-Path $rootDir ("bench\baseline.{0}.xml" -f $Platform)
$doc.Save($platformPath)
Write-Host ("Saved {0}" -f $platformPath)

if ($UpdateCompatibilitySnapshot) {
    $compatPath = Join-Path $rootDir "bench\baseline.xml"
    $doc.Save($compatPath)
    Write-Host ("Saved {0}" -f $compatPath)
}
