$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $repoRoot

$releaseDir = Join-Path $repoRoot "release"
if (Test-Path $releaseDir) {
    Remove-Item -Recurse -Force $releaseDir
}

$targets = @(
    @{ Goos = "windows"; Goarch = "amd64"; Ext = ".exe" },
    @{ Goos = "linux"; Goarch = "amd64"; Ext = "" }
)

$repoOwner = "radu103"
$repoName = "git-enterprise-hooks"

function Resolve-ReleaseVersion {
    if (-not $env:RELEASE_VERSION -or $env:RELEASE_VERSION.Trim() -eq "") {
        throw "RELEASE_VERSION is required. Set it before running build-release.ps1 (example: `$env:RELEASE_VERSION='0.0.1')."
    }

    return $env:RELEASE_VERSION.TrimStart("v").Trim()
}

$version = Resolve-ReleaseVersion

foreach ($target in $targets) {
    $osFolder = Join-Path $releaseDir $target.Goos
    New-Item -ItemType Directory -Force -Path $osFolder | Out-Null

    $binaryName = "git-enterprise-hooks-$($target.Goos)-$($target.Goarch)$($target.Ext)"
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

$windowsExeName = "git-enterprise-hooks-windows-amd64.exe"
$windowsBinary = Join-Path $releaseDir "windows\$windowsExeName"
if (Test-Path $windowsBinary) {
    $legacyZip = Join-Path $releaseDir "windows\git-enterprise-hooks-windows-amd64.zip"
    if (Test-Path $legacyZip) {
        Remove-Item -Force $legacyZip
    }

    $hash = (Get-FileHash -Algorithm SHA256 -Path $windowsBinary).Hash.ToLower()
    $scoopManifestPath = Join-Path $releaseDir "windows\git-enterprise-hooks.json"
    $downloadUrl = "https://github.com/$repoOwner/$repoName/releases/download/v$version/$windowsExeName"

    $manifest = [ordered]@{
        version     = $version
        description = "CLI helper that installs a pre-commit hook and guides task-based commit message creation."
        homepage    = "https://github.com/$repoOwner/$repoName"
        license     = "MIT"
        architecture = [ordered]@{
            "64bit" = [ordered]@{
                url  = $downloadUrl
                hash = $hash
            }
        }
        # Expose a stable command name in Scoop regardless of the release filename.
        # Use unary comma to keep the inner pair as a single nested array item.
        bin = @(, @($windowsExeName, "git-enterprise-hooks"))
        autoupdate = [ordered]@{
            architecture = [ordered]@{
                "64bit" = [ordered]@{
                    url = "https://github.com/$repoOwner/$repoName/releases/download/v`$version/$windowsExeName"
                }
            }
        }
    }

    $manifest | ConvertTo-Json -Depth 8 | Set-Content -Path $scoopManifestPath -Encoding UTF8
    Write-Host "Scoop manifest generated: $scoopManifestPath"
}

Write-Host "Release artifacts generated under: $releaseDir"
