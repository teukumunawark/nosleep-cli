param(
    [string] $Version = "latest",
    [string] $InstallDir = "$env:LOCALAPPDATA\Programs\NoSleep"
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

    $updated = $false
    $exists = $entries | ForEach-Object {
        $entry = $_
        $expandedEntry = [Environment]::ExpandEnvironmentVariables($entry).TrimEnd("\")
        if ($expandedEntry.Equals($expandedPathEntry, [StringComparison]::OrdinalIgnoreCase)) {
            if (-not $entry.Equals($PathEntry, [StringComparison]::Ordinal)) {
                $updated = $true
                return $PathEntry
            }

            return $entry
        }

        return $entry
    }

    $hasEntry = $exists | Where-Object {
        [Environment]::ExpandEnvironmentVariables($_).TrimEnd("\").Equals(
            $expandedPathEntry,
            [StringComparison]::OrdinalIgnoreCase
        )
    } | Select-Object -First 1

    if ($hasEntry) {
        if ($updated) {
            [Environment]::SetEnvironmentVariable("Path", ($exists -join ";"), "User")
            Write-Host "Updated User Path entry to $PathEntry"
            return
        }

        Write-Host "User Path already contains $PathEntry"
        return
    }

    $newPath = (($exists + $PathEntry) -join ";")
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    Write-Host "Added $PathEntry to User Path"
}

function Repair-InstallDirCasing {
    param(
        [Parameter(Mandatory = $true)]
        [string] $Path
    )

    $parent = Split-Path -Parent $Path
    $leaf = Split-Path -Leaf $Path
    if (-not (Test-Path -LiteralPath $parent)) {
        return
    }

    $existing = Get-ChildItem -LiteralPath $parent -Directory -Force | Where-Object {
        $_.Name.Equals($leaf, [StringComparison]::OrdinalIgnoreCase)
    } | Select-Object -First 1

    if ($existing -and $existing.Name -cne $leaf) {
        $temporaryPath = Join-Path $parent ("." + $leaf + "-rename-" + [Guid]::NewGuid().ToString("N"))
        Rename-Item -LiteralPath $existing.FullName -NewName (Split-Path -Leaf $temporaryPath)
        Rename-Item -LiteralPath $temporaryPath -NewName $leaf
    }
}

function Get-StatePath {
    $base = $env:LOCALAPPDATA
    if (-not $base) {
        $base = [Environment]::GetFolderPath("LocalApplicationData")
    }

    return Join-Path $base "NoSleepCLI\state.json"
}

function Get-ProcessPath {
    param(
        [Parameter(Mandatory = $true)]
        $Process
    )

    try {
        if ($Process.Path) {
            return $Process.Path
        }
    } catch {
    }

    try {
        return $Process.MainModule.FileName
    } catch {
        return ""
    }
}

function Test-ProcessMatchesPath {
    param(
        [Parameter(Mandatory = $true)]
        $Process,

        [Parameter(Mandatory = $true)]
        [string] $Path
    )

    $processPath = Get-ProcessPath -Process $Process
    if (-not $processPath) {
        return $false
    }

    return $processPath.Equals($Path, [StringComparison]::OrdinalIgnoreCase)
}

function Assert-NoSleepNotRunning {
    param(
        [Parameter(Mandatory = $true)]
        [string] $TargetPath
    )

    $statePath = Get-StatePath
    if (Test-Path -LiteralPath $statePath) {
        try {
            $state = Get-Content -LiteralPath $statePath -Raw | ConvertFrom-Json
            if ($state.pid) {
                $process = Get-Process -Id ([int] $state.pid) -ErrorAction SilentlyContinue
                if ($process -and (Test-ProcessMatchesPath -Process $process -Path $state.executable)) {
                    throw "NoSleep is currently running. Run 'nosleep stop' before updating."
                }
            }
        } catch {
            if ($_.Exception.Message -like "NoSleep is currently running.*") {
                throw
            }
        }
    }

    $target = [IO.Path]::GetFullPath($TargetPath)
    $running = Get-Process -Name "nosleep" -ErrorAction SilentlyContinue | Where-Object {
        Test-ProcessMatchesPath -Process $_ -Path $target
    } | Select-Object -First 1

    if ($running) {
        throw "NoSleep is currently running. Run 'nosleep stop' before updating."
    }
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

    $targetPath = Join-Path $InstallDir "nosleep.exe"
    Assert-NoSleepNotRunning -TargetPath $targetPath

    Repair-InstallDirCasing -Path $InstallDir
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Copy-Item -LiteralPath $binaryPath -Destination $targetPath -Force

    Write-Host "Installed $targetPath"
    Write-Host "Version: $($release.tag_name)"
    Write-Host "Verified SHA-256: $actualHash"

    Add-UserPathEntry -PathEntry $InstallDir

    Write-Host "Installation complete! Open a new terminal before running nosleep." -ForegroundColor Green
} finally {
    Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue
}
