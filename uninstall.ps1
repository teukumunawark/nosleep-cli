param(
    [string] $InstallDir = "$env:LOCALAPPDATA\Programs\NoSleep"
)

$ErrorActionPreference = "Stop"

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
                    throw "NoSleep is currently running. Run 'nosleep stop' before uninstalling."
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
        throw "NoSleep is currently running. Run 'nosleep stop' before uninstalling."
    }
}

function Remove-UserPathEntry {
    param(
        [Parameter(Mandatory = $true)]
        [string] $PathEntry
    )

    $expandedPathEntry = [Environment]::ExpandEnvironmentVariables($PathEntry).TrimEnd("\")
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if (-not $userPath) {
        return
    }

    $entries = $userPath -split ";" | Where-Object {
        $_ -and -not [Environment]::ExpandEnvironmentVariables($_).TrimEnd("\").Equals(
            $expandedPathEntry,
            [StringComparison]::OrdinalIgnoreCase
        )
    }

    [Environment]::SetEnvironmentVariable("Path", ($entries -join ";"), "User")
    Write-Host "Removed $PathEntry from User Path"
}

function Get-LegacyInstallDir {
    return Join-Path $env:LOCALAPPDATA "Programs\nosleep"
}

function Remove-InstallDir {
    param(
        [Parameter(Mandatory = $true)]
        [string] $Path
    )

    $targetPath = Join-Path $Path "nosleep.exe"
    Assert-NoSleepNotRunning -TargetPath $targetPath

    if (Test-Path -LiteralPath $targetPath) {
        Remove-Item -LiteralPath $targetPath -Force
        Write-Host "Removed $targetPath"
    }

    if (Test-Path -LiteralPath $Path) {
        $remaining = Get-ChildItem -LiteralPath $Path -Force
        if (-not $remaining) {
            Remove-Item -LiteralPath $Path -Force
            Write-Host "Removed $Path"
        }
    }

    Remove-UserPathEntry -PathEntry $Path
}

$installDirs = @($InstallDir)
$legacyInstallDir = Get-LegacyInstallDir
$normalizedInstallDir = [IO.Path]::GetFullPath($InstallDir).TrimEnd("\")
$normalizedLegacyDir = [IO.Path]::GetFullPath($legacyInstallDir).TrimEnd("\")

if (-not $normalizedInstallDir.Equals($normalizedLegacyDir, [StringComparison]::Ordinal)) {
    $installDirs += $legacyInstallDir
}

foreach ($dir in $installDirs) {
    Remove-InstallDir -Path $dir
}

$statePath = Get-StatePath
if (Test-Path -LiteralPath $statePath) {
    Remove-Item -LiteralPath $statePath -Force
    Write-Host "Removed $statePath"
}

$stateDir = Split-Path -Parent $statePath
if (Test-Path -LiteralPath $stateDir) {
    $remaining = Get-ChildItem -LiteralPath $stateDir -Force
    if (-not $remaining) {
        Remove-Item -LiteralPath $stateDir -Force
        Write-Host "Removed $stateDir"
    }
}

Write-Host "Uninstall complete! Open a new terminal to refresh PATH." -ForegroundColor Green
