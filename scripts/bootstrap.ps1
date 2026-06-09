param(
    [switch]$NoWinget,
    [switch]$NoWails,
    [switch]$NoFrontend,
    [switch]$NoTidy
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $Root

function HasCommand($Name) {
    return $null -ne (Get-Command $Name -ErrorAction SilentlyContinue)
}

if (-not $NoWinget -and (HasCommand winget)) {
    if (-not (HasCommand go)) {
        winget install --id GoLang.Go -e --accept-package-agreements --accept-source-agreements
    }
    if (-not (HasCommand node)) {
        winget install --id OpenJS.NodeJS.LTS -e --accept-package-agreements --accept-source-agreements
    }
    if (-not (HasCommand git)) {
        winget install --id Git.Git -e --accept-package-agreements --accept-source-agreements
    }
}

if (-not (HasCommand go)) {
    throw "Go is not installed or not on PATH. Install Go 1.25+ and rerun scripts\bootstrap.bat."
}
if (-not (HasCommand npm)) {
    throw "npm is not installed or not on PATH. Install Node.js LTS and rerun scripts\bootstrap.bat."
}

New-Item -ItemType Directory -Force -Path ".build" | Out-Null
New-Item -ItemType Directory -Force -Path "frontend/dist" | Out-Null
New-Item -ItemType File -Force -Path "frontend/dist/.gitkeep" | Out-Null
"WAILS_WEBKIT_TAG=" | Set-Content ".build/webkit.env"

if (-not $NoWails) {
    go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0
}
if (-not $NoTidy) {
    go mod tidy
    go mod download
}
if (-not $NoFrontend) {
    Push-Location frontend
    npm ci
    Pop-Location
}

if (HasCommand wails) {
    wails doctor
} else {
    Write-Host "Wails CLI installed into GOPATH\bin. Add it to PATH if 'wails' is unavailable."
}

Write-Host "Bootstrap complete. Try: build.bat check"
