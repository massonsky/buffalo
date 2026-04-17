param(
    [Parameter(Mandatory=$true)]
    [Alias("m")]
    [string]$Message
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Write-Host "=== git add -A ===" -ForegroundColor Cyan
git add -A

Write-Host "`n=== git status ===" -ForegroundColor Cyan
git status

Write-Host "`n=== git commit ===" -ForegroundColor Cyan
git commit -m $Message
if ($LASTEXITCODE -ne 0) { Write-Host "Commit failed" -ForegroundColor Red; exit 1 }

Write-Host "`n=== git push ===" -ForegroundColor Cyan
git push -u origin HEAD

Write-Host "`nDone (tag will be created by CI after successful build)" -ForegroundColor Green
