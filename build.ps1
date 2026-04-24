$ErrorActionPreference = "Stop"

$repoRoot = $PSScriptRoot
if (-not $repoRoot) { $repoRoot = Get-Location }

$outputDir = "C:\Tools\nosleep"
$outputFile = Join-Path $outputDir "nosleep.exe"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "Go is not installed or not available in PATH."
}

if (-not (Test-Path -LiteralPath $outputDir)) {
    try {
        New-Item -ItemType Directory -Path $outputDir -Force | Out-Null
    } catch {
        throw "Failed to create directory $outputDir. Please run as Administrator or create it manually."
    }
}

Write-Host "Building nosleep to $outputFile" -ForegroundColor Cyan
Push-Location $repoRoot
try {
    & go build -o $outputFile .\cmd\nosleep
} finally {
    Pop-Location
}

if ($LASTEXITCODE -ne 0) {
    throw "Build failed."
}

Write-Host "Build complete: $outputFile" -ForegroundColor Green
