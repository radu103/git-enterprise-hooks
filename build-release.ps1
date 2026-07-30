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

$repoOwner = "radu103"
$repoName = "git-enterprise-hooks"

function Resolve-ReleaseVersion {
    if ($env:RELEASE_VERSION -and $env:RELEASE_VERSION.Trim() -ne "") {
        return $env:RELEASE_VERSION.Trim()
    }

    $tag = git describe --tags --exact-match 2>$null
    if ($LASTEXITCODE -eq 0 -and $tag) {
        return $tag.TrimStart("v")
    }

    return "0.0.0-dev"
}

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

$version = Resolve-ReleaseVersion
$windowsBinary = Join-Path $releaseDir "windows\git-enterprise-hooks-amd64.exe"
if (Test-Path $windowsBinary) {
    $hash = (Get-FileHash -Algorithm SHA256 -Path $windowsBinary).Hash.ToLower()
    $scoopManifestPath = Join-Path $releaseDir "windows\git-enterprise-hooks.json"
    $downloadUrl = "https://github.com/$repoOwner/$repoName/releases/download/v$version/git-enterprise-hooks-amd64.exe"

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
        bin = @("git-enterprise-hooks-amd64.exe")
        autoupdate = [ordered]@{
            architecture = [ordered]@{
                "64bit" = [ordered]@{
                    url = "https://github.com/$repoOwner/$repoName/releases/download/v`$version/git-enterprise-hooks-amd64.exe"
                }
            }
        }
    }

    $manifest | ConvertTo-Json -Depth 8 | Set-Content -Path $scoopManifestPath -Encoding UTF8
    Write-Host "Scoop manifest generated: $scoopManifestPath"
}

Write-Host "Release artifacts generated under: $releaseDir"
