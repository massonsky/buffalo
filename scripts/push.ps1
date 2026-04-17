param(
    [Parameter(Mandatory=$true)]
    [Alias("m")]
    [string]$Message,

    [Alias("maj")]
    [int]$Major = 1,

    [Alias("min")]
    [int]$Minor = 32
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

$hash = git rev-parse --short HEAD
$tag = "v$Major.$Minor.$hash"

Write-Host "`n=== git tag $tag ===" -ForegroundColor Cyan
git tag $tag

Write-Host "`n=== git push ===" -ForegroundColor Cyan
git push
git push --tags

Write-Host "`nDone: $tag" -ForegroundColor Green
