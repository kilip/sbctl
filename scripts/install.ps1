# sbctl installer for Windows
# Usage: irm https://raw.githubusercontent.com/kilip/sbctl/main/scripts/install.ps1 | iex

param(
    [string]$Version = "",
    [string]$InstallDir = "$env:USERPROFILE\.local\bin"
)

$ErrorActionPreference = "Stop"

$Repo   = "kilip/sbctl"
$Binary = "sbctl"

function Write-Info    { param($msg) Write-Host "[sbctl] $msg" -ForegroundColor Cyan }
function Write-Success { param($msg) Write-Host "[sbctl] $msg" -ForegroundColor Green }
function Write-Warn    { param($msg) Write-Host "[sbctl] $msg" -ForegroundColor Yellow }
function Write-Err     { param($msg) Write-Host "[sbctl] $msg" -ForegroundColor Red; exit 1 }

function Get-LatestVersion {
    try {
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
        return $response.tag_name
    } catch {
        Write-Err "Could not fetch latest version from GitHub: $_"
    }
}

function Get-Arch {
    switch ($env:PROCESSOR_ARCHITECTURE) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        default { Write-Err "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
    }
}

function Install-Binary {
    param($Arch, $Ver)

    $verNoV   = $Ver -replace '^v', ''
    $filename = "${Binary}-${verNoV}-windows-${Arch}.zip"
    $url      = "https://github.com/$Repo/releases/download/$Ver/$filename"
    $tmpDir   = Join-Path $env:TEMP "sbctl_install_$(Get-Random)"

    New-Item -ItemType Directory -Path $tmpDir | Out-Null

    Write-Info "Downloading $Binary $Ver (windows/$Arch)..."
    try {
        Invoke-WebRequest -Uri $url -OutFile "$tmpDir\$filename" -UseBasicParsing
    } catch {
        Remove-Item -Recurse -Force $tmpDir
        Write-Err "Failed to download: $url`n$_"
    }

    Write-Info "Extracting..."
    Expand-Archive -Path "$tmpDir\$filename" -DestinationPath $tmpDir -Force

    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir | Out-Null
    }

    Move-Item -Force "$tmpDir\$Binary.exe" "$InstallDir\$Binary.exe"
    Remove-Item -Recurse -Force $tmpDir
}

function Add-ToPath {
    $currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($currentPath -notlike "*$InstallDir*") {
        Write-Warn "$InstallDir is not in your PATH."
        Write-Warn "Adding it now to user PATH..."
        [Environment]::SetEnvironmentVariable("PATH", "$currentPath;$InstallDir", "User")
        $env:PATH = "$env:PATH;$InstallDir"
        Write-Success "PATH updated. Restart your terminal to apply."
    }
}

function Main {
    Write-Info "Installing sbctl..."

    $arch = Get-Arch
    $ver  = if ($Version -ne "") { $Version } else { Get-LatestVersion }

    Write-Info "Version : $ver"
    Write-Info "Arch    : $arch"
    Write-Info "Target  : $InstallDir\$Binary.exe"

    Install-Binary -Arch $arch -Ver $ver

    Write-Success "$Binary $ver installed to $InstallDir\$Binary.exe"

    Add-ToPath

    Write-Info "Running: sbctl setup"
    Write-Host ""
    & "$InstallDir\$Binary.exe" setup
}

Main
