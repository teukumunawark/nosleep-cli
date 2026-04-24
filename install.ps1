param(
    [string] $Version = "latest",
    [string] $InstallDir = "$env:LOCALAPPDATA\Programs\nosleep",
    [switch] $AddToPath
)

$ErrorActionPreference = "Stop"

$repo = "teukumunawark/nosleep-cli"
$apiBase = "https://api.github.com/repos/$repo"

function Get-Release {
    if ($Version -eq "latest") {
        return Invoke-RestMethod -Uri "$apiBase/releases/latest" -Headers @{ "User-Agent" = "nosleep-installer" }
    }

    return Invoke-RestMethod -Uri "$apiBase/releases/tags/$Version" -Headers @{ "User-Agent" = "nosleep-installer" }
}

function Get-Asset {
    param(
        [Parameter(Mandatory = $true)]
        $Release,

        [Parameter(Mandatory = $true)]
        [string] $Name
    )

    $asset = $Release.assets | Where-Object { $_.name -eq $Name } | Select-Object -First 1
    if (-not $asset) {
        throw "Release asset not found: $Name"
    }

    return $asset
}

function Get-WindowsArch {
    $arch = $env:PROCESSOR_ARCHITEW6432
    if (-not $arch) {
        $arch = $env:PROCESSOR_ARCHITECTURE
    }

    switch -Regex ($arch) {
        "ARM64" { return "arm64" }
        "AMD64|x86_64" { return "amd64" }
        default { throw "Unsupported Windows architecture: $arch" }
    }
}

function Add-UserPathEntry {
    param(
        [Parameter(Mandatory = $true)]
        [string] $PathEntry
    )

    $expandedPathEntry = [Environment]::ExpandEnvironmentVariables($PathEntry).TrimEnd("\")
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $entries = @()

    if ($userPath) {
        $entries = $userPath -split ";" | Where-Object { $_ }
    }

    $exists = $entries | Where-Object {
        [Environment]::ExpandEnvironmentVariables($_).TrimEnd("\").Equals(
            $expandedPathEntry,
            [StringComparison]::OrdinalIgnoreCase
        )
    } | Select-Object -First 1

    if ($exists) {
        Write-Host "User Path already contains $PathEntry"
        return
    }

    $newPath = (($entries + $PathEntry) -join ";")
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    Write-Host "Added $PathEntry to User Path"
}

$arch = Get-WindowsArch
$assetName = "nosleep-windows-$arch.exe"

Write-Host "Resolving release $Version from github.com/$repo"
$release = Get-Release
$binaryAsset = Get-Asset -Release $release -Name $assetName
$checksumAsset = Get-Asset -Release $release -Name "checksums.txt"

$tempDir = Join-Path ([IO.Path]::GetTempPath()) ("nosleep-install-" + [Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

try {
    $binaryPath = Join-Path $tempDir $assetName
    $checksumPath = Join-Path $tempDir "checksums.txt"

    Write-Host "Downloading $assetName"
    Invoke-WebRequest -Uri $binaryAsset.browser_download_url -OutFile $binaryPath
    Invoke-WebRequest -Uri $checksumAsset.browser_download_url -OutFile $checksumPath

    $checksumLine = Get-Content $checksumPath | Where-Object { $_ -match "\s+$([Regex]::Escape($assetName))$" } | Select-Object -First 1
    if (-not $checksumLine) {
        throw "Checksum entry not found for $assetName"
    }

    $expectedHash = ($checksumLine -split "\s+")[0].ToLowerInvariant()
    $actualHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $binaryPath).Hash.ToLowerInvariant()

    if ($actualHash -ne $expectedHash) {
        throw "Checksum verification failed for $assetName"
    }

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    $targetPath = Join-Path $InstallDir "nosleep.exe"
    Copy-Item -LiteralPath $binaryPath -Destination $targetPath -Force

    Write-Host "Installed $targetPath"
    Write-Host "Verified SHA-256: $actualHash"

    if ($AddToPath) {
        Add-UserPathEntry -PathEntry $InstallDir
        Write-Host "Open a new terminal before running nosleep."
    } else {
        Write-Host "Add this directory to User Path to run nosleep from any terminal:"
        Write-Host $InstallDir
    }
} finally {
    Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue
}
