function Get-BenchManifest {
    param(
        [Parameter(Mandatory = $true)]
        [string]$RootDir
    )

    $manifestPath = Join-Path $RootDir "bench\manifest.json"
    if (-not (Test-Path $manifestPath)) {
        throw ("benchmark manifest not found: {0}" -f $manifestPath)
    }

    try {
        return Get-Content -Raw $manifestPath | ConvertFrom-Json -ErrorAction Stop
    } catch {
        throw ("failed to load benchmark manifest '{0}': {1}" -f $manifestPath, $_.Exception.Message)
    }
}

function Get-BenchEntries {
    param(
        [Parameter(Mandatory = $true)]
        [object]$Manifest
    )

    $entries = @()
    foreach ($entry in $Manifest.benchmarks) {
        $entries += @{
            Name = [string]$entry.name
            Path = [string]$entry.path
            LLVMRatioTarget = [double]$entry.llvm_ratio_target
        }
    }
    return $entries
}

function Get-BenchNames {
    param(
        [Parameter(Mandatory = $true)]
        [object]$Manifest
    )

    $names = @()
    foreach ($entry in $Manifest.benchmarks) {
        $names += [string]$entry.name
    }
    return $names
}

function Get-BenchThresholdPercent {
    param(
        [Parameter(Mandatory = $true)]
        [object]$Manifest
    )

    return [int]$Manifest.baseline_threshold_percent
}
