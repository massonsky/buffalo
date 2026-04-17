param(
    [Parameter(Mandatory=$true)]
    [Alias("m")]
    [string]$Message,

    [Alias("maj")]
    [int]$Major = 1,

    [Alias("min")]
    [int]$Minor = 32,

    [Alias("pat")]
    [int]$Patch = -1
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# Auto-increment patch from latest tag if not specified
if ($Patch -lt 0) {
    $prefix = "v$Major.$Minor."
    $latest = git tag -l "${prefix}*" --sort=-v:refname 2>$null | Select-Object -First 1
    if ($latest -and $latest -match "^v\d+\.\d+\.(\d+)$") {
        $Patch = [int]$Matches[1] + 1
    } else {
        $Patch = 0
    }
}

Write-Host "=== git add -A ===" -ForegroundColor Cyan
git add -A

Write-Host "`n=== git status ===" -ForegroundColor Cyan
git status

Write-Host "`n=== git commit ===" -ForegroundColor Cyan
git commit -m $Message
if ($LASTEXITCODE -ne 0) { Write-Host "Commit failed" -ForegroundColor Red; exit 1 }

$tag = "v$Major.$Minor.$Patch"

Write-Host "`n=== git tag $tag ===" -ForegroundColor Cyan
git tag $tag

Write-Host "`n=== git push ===" -ForegroundColor Cyan
git push -u origin HEAD
git push origin $tag

Write-Host "`nDone: $tag" -ForegroundColor Green
