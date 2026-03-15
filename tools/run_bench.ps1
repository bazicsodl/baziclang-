param(
    [int]$Iterations = 10000000,
    [int]$Runs = 3
)

$root = Get-Location
$bench = Join-Path $root "examples\bench\perf.bz"
$bazic = Join-Path $root "bin\bazic.exe"
$backends = @("go", "llvm")

foreach ($backend in $backends) {
    Write-Host "Backend: $backend"
    for ($i = 1; $i -le $Runs; $i++) {
        Write-Host (" Run {0}/{1}" -f $i, $Runs)
        $env:BAZIC_BACKEND = $backend
        $env:BAZIC_ITERS = $Iterations
        Measure-Command { & $bazic run $bench } | Select-Object TotalMilliseconds |
            ForEach-Object { Write-Host ("  Cmd ms: {0:N0}" -f $_.TotalMilliseconds) }
    }
    Write-Host ""
}

Remove-Item Env:BAZIC_BACKEND
Remove-Item Env:BAZIC_ITERS
