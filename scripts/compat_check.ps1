param(
  [string]$Root = ".",
  [string]$Target = "v1.0"
)

Set-Location $Root

$versionFile = "LANGUAGE.md"
if (-not (Test-Path $versionFile)) {
  Write-Error "LANGUAGE.md not found"
  exit 1
}

$hasDraft = Select-String -Path $versionFile -Pattern "Draft" -Quiet
if ($Target -eq "v1.0" -and $hasDraft) {
  Write-Error "compat check failed: LANGUAGE.md still marked Draft"
  exit 1
}

Write-Host "compat check passed for $Target"
