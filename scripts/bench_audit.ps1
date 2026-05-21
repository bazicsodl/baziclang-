param(
    [switch]$RequireAll
)

$scriptPath = $MyInvocation.MyCommand.Path
$rootDir = Split-Path -Parent $scriptPath
$rootDir = Split-Path -Parent $rootDir
$benchDir = Join-Path $rootDir "bench"

$platformFiles = @(
    @{ Platform = "windows"; Path = Join-Path $benchDir "baseline.windows.xml" },
    @{ Platform = "linux"; Path = Join-Path $benchDir "baseline.linux.xml" },
    @{ Platform = "macos"; Path = Join-Path $benchDir "baseline.macos.xml" }
)

function Read-BaselineXml([string]$path) {
    try {
        return [xml](Get-Content -Raw $path -ErrorAction Stop)
    } catch {
        Write-Error ("Failed to parse baseline XML: {0}" -f $path)
        return $null
    }
}

function Validate-BaselineDoc([xml]$doc, [string]$expectedPlatform, [string]$path) {
    if ($null -eq $doc) {
        return $false
    }
    $baseline = $doc.benchmarks.baseline
    if (-not $baseline) {
        Write-Error ("Missing <baseline> node: {0}" -f $path)
        return $false
    }
    if ([string]$baseline.platform -ne $expectedPlatform) {
        Write-Error ("Baseline platform mismatch in {0}: expected '{1}', got '{2}'" -f $path, $expectedPlatform, [string]$baseline.platform)
        return $false
    }
    if (-not $baseline.go -or -not $baseline.llvm) {
        Write-Error ("Baseline missing go/llvm sections: {0}" -f $path)
        return $false
    }
    return $true
}

$failed = $false
$available = @{}
foreach ($item in $platformFiles) {
    if (-not (Test-Path $item.Path)) {
        if ($RequireAll) {
            Write-Error ("Missing required platform baseline: {0}" -f $item.Path)
            $failed = $true
        } else {
            Write-Host ("skip {0}: missing" -f $item.Platform)
        }
        continue
    }
    $doc = Read-BaselineXml $item.Path
    if (-not (Validate-BaselineDoc $doc $item.Platform $item.Path)) {
        $failed = $true
        continue
    }
    $available[$item.Platform] = $doc
    Write-Host ("ok   {0}: {1}" -f $item.Platform, $item.Path)
}

$compatPath = Join-Path $benchDir "baseline.xml"
if (Test-Path $compatPath) {
    $compatDoc = Read-BaselineXml $compatPath
    if ($null -eq $compatDoc) {
        $failed = $true
    } else {
        $compatBaseline = $compatDoc.benchmarks.baseline
        if (-not $compatBaseline) {
            Write-Error ("Missing <baseline> node: {0}" -f $compatPath)
            $failed = $true
        } else {
            $compatPlatform = [string]$compatBaseline.platform
            if (-not $compatPlatform) {
                Write-Error ("Compatibility baseline missing platform attribute: {0}" -f $compatPath)
                $failed = $true
            } elseif ($available.ContainsKey($compatPlatform)) {
                $platformDoc = $available[$compatPlatform]
                if ($platformDoc.OuterXml -ne $compatDoc.OuterXml) {
                    Write-Error ("Compatibility baseline does not match bench/baseline.{0}.xml" -f $compatPlatform)
                    $failed = $true
                } else {
                    Write-Host ("ok   compatibility snapshot matches {0}" -f $compatPlatform)
                }
            } else {
                Write-Host ("skip compatibility snapshot match: no committed platform baseline for {0}" -f $compatPlatform)
            }
        }
    }
} elseif ($RequireAll) {
    Write-Error ("Missing compatibility baseline: {0}" -f $compatPath)
    $failed = $true
}

if ($failed) {
    exit 1
}
