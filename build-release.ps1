$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $repoRoot

$releaseDir = Join-Path $repoRoot "release"
if (Test-Path $releaseDir) {
    Remove-Item -Recurse -Force $releaseDir
}

$targets = @(
    @{ Goos = "windows"; Goarch = "amd64"; Ext = ".exe" },
    @{ Goos = "linux"; Goarch = "amd64"; Ext = "" },
    @{ Goos = "darwin"; Goarch = "amd64"; Ext = "" }
)

foreach ($target in $targets) {
    $osFolder = Join-Path $releaseDir $target.Goos
    New-Item -ItemType Directory -Force -Path $osFolder | Out-Null

    $binaryName = "git-enterprise-hooks-$($target.Goarch)$($target.Ext)"
    $outputPath = Join-Path $osFolder $binaryName

    Write-Host "Building $($target.Goos)/$($target.Goarch) -> $outputPath"

    $env:GOOS = $target.Goos
    $env:GOARCH = $target.Goarch
    $env:CGO_ENABLED = "0"

    go build -trimpath -ldflags "-s -w" -o $outputPath .
}

Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue

Write-Host "Release artifacts generated under: $releaseDir"
