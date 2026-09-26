# BuildShare CLI Installer for Windows
# https://buildshare.in
#
# Usage:
#   irm https://buildshare.in/install.ps1 | iex
#
# Environment variables:
#   $env:BUILDSHARE_INSTALL_DIR  Custom directory to install the binary into
#   $env:BUILDSHARE_VERSION      Specific version/tag to install (default: latest)
#   $env:GITHUB_OWNER            GitHub repository owner (default: vishalkumardev)
#   $env:GITHUB_REPO             GitHub repository name (default: cli)

$ErrorActionPreference = 'Stop'

# Configuration
$Owner = if ($env:GITHUB_OWNER) { $env:GITHUB_OWNER } else { "vishalkumardev" }
$Repo = if ($env:GITHUB_REPO) { $env:GITHUB_REPO } else { "cli" }
$BinaryName = "buildshare.exe"

Write-Host "BuildShare CLI Installer" -ForegroundColor Cyan
Write-Host ""

# 1. Architecture detection
$is64BitOS = [Environment]::Is64BitOperatingSystem
$procArch = $env:PROCESSOR_ARCHITECTURE

if (-not $is64BitOS) {
    Write-Error "BuildShare CLI requires a 64-bit operating system."
    exit 1
}

$platformArch = "amd64"
if ($procArch -eq "ARM64") {
    $platformArch = "arm64"
}

Write-Host "Detected platform: Windows"
Write-Host "Detected architecture: $platformArch"
Write-Host ""

# 2. Existing installation detection
$existingBin = Get-Command "buildshare" -ErrorAction SilentlyContinue
if ($existingBin) {
    Write-Host "Found existing installation at: $($existingBin.Source)"
}

# 3. Determine target directory
$installDir = if ($env:BUILDSHARE_INSTALL_DIR) {
    $env:BUILDSHARE_INSTALL_DIR
} elseif ($existingBin) {
    Split-Path -Parent $existingBin.Source
} else {
    Join-Path $env:LOCALAPPDATA "Programs\buildshare\bin"
}

# 4. Resolve download URL
Write-Host "Downloading BuildShare CLI..."

# Resolve latest release tag
$tag = if ($env:BUILDSHARE_VERSION) { $env:BUILDSHARE_VERSION } else { "" }
if (-not $tag) {
    try {
        $latestUrl = "https://github.com/$Owner/$Repo/releases/latest"
        $req = [System.Net.WebRequest]::Create($latestUrl)
        $req.AllowAutoRedirect = $false
        $resp = $req.GetResponse()
        $location = $resp.GetResponseHeader("Location")
        $resp.Close()
        if ($location) {
            $tag = $location.Split('/')[-1]
        }
    } catch {
        # Fallback to GitHub API
        try {
            $apiRelease = Invoke-RestMethod -Uri "https://api.github.com/repos/$Owner/$Repo/releases/latest" -UseBasicParsing
            if ($apiRelease.tag_name) {
                $tag = $apiRelease.tag_name
            }
        } catch {}
    }
}

# Setup temp directory
$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("buildshare-install-" + [System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

try {
    # Candidate URLs to support versioned archives and standalone releases
    $candidateUrls = @()
    if ($tag) {
        $candidateUrls += "https://github.com/$Owner/$Repo/releases/latest/download/buildshare_${tag}_windows_${platformArch}.zip"
        $candidateUrls += "https://github.com/$Owner/$Repo/releases/download/$tag/buildshare_${tag}_windows_${platformArch}.zip"
    }
    $candidateUrls += "https://github.com/$Owner/$Repo/releases/latest/download/buildshare_windows_${platformArch}.zip"
    $candidateUrls += "https://github.com/$Owner/$Repo/releases/latest/download/buildshare-windows-${platformArch}.zip"
    $candidateUrls += "https://github.com/$Owner/$Repo/releases/latest/download/buildshare-windows-${platformArch}.exe"

    $downloadedFile = $null
    $isArchive = $false

    # Ensure TLS 1.2+ is enabled
    [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor [System.Net.SecurityProtocolType]::Tls12

    foreach ($url in $candidateUrls) {
        $fileName = Split-Path -Leaf $url
        $dest = Join-Path $tempDir $fileName
        try {
            Invoke-WebRequest -Uri $url -OutFile $dest -UseBasicParsing -ErrorAction Stop
            if ((Test-Path $dest) -and ((Get-Item $dest).Length -gt 0)) {
                $downloadedFile = $dest
                if ($fileName.EndsWith(".zip")) {
                    $isArchive = $true
                }
                break
            }
        } catch {
            # Try next candidate
        }
    }

    if (-not $downloadedFile) {
        throw "Failed to download BuildShare CLI binary for windows-$platformArch. Please verify that a release exists at https://github.com/$Owner/$Repo/releases"
    }

    # Extract or locate executable
    $extractedExe = $null
    if ($isArchive) {
        $extractDir = Join-Path $tempDir "extracted"
        Expand-Archive -Path $downloadedFile -DestinationPath $extractDir -Force
        $found = Get-ChildItem -Path $extractDir -Filter "buildshare*.exe" -Recurse | Select-Object -First 1
        if ($found) {
            $extractedExe = $found.FullName
        } else {
            throw "Executable 'buildshare.exe' not found inside downloaded archive."
        }
    } else {
        $extractedExe = $downloadedFile
    }

    # 5. Install binary
    Write-Host "Installing BuildShare CLI..."
    if (-not (Test-Path $installDir)) {
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    }

    $finalExe = Join-Path $installDir $BinaryName
    $tempExe = Join-Path $installDir "$BinaryName.tmp"

    # Copy to temp file first, then atomic move
    Copy-Item -Path $extractedExe -Destination $tempExe -Force
    Move-Item -Path $tempExe -Destination $finalExe -Force

    # 6. Verify installation
    $versionOutput = ""
    try {
        $verRes = & "$finalExe" version 2>$null
        if ($verRes) {
            $match = ($verRes | Select-String "BuildShare CLI\s+(\S+)").Matches
            if ($match) {
                $versionOutput = $match.Groups[1].Value
            }
        }
    } catch {}

    if (-not $versionOutput) {
        try {
            $verRes = & "$finalExe" --version 2>$null
            if ($verRes) {
                $versionOutput = ($verRes -split '\s+')[-1]
            }
        } catch {}
    }

    if (-not $versionOutput -and $tag) {
        $versionOutput = $tag
    }
    if (-not $versionOutput) {
        $versionOutput = "installed"
    }

    # 7. Update PATH if needed
    $pathEnv = [Environment]::GetEnvironmentVariable("Path", "User")
    $paths = if ($pathEnv) { $pathEnv -split ';' } else { @() }
    $inPath = $false
    foreach ($p in $paths) {
        if ($p.TrimEnd('\') -eq $installDir.TrimEnd('\')) {
            $inPath = $true
            break
        }
    }

    if (-not $inPath) {
        $newPath = if ($pathEnv) { "$pathEnv;$installDir" } else { $installDir }
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        # Also update current session PATH
        $env:PATH = "$env:PATH;$installDir"
    }

    Write-Host ""
    Write-Host "✓ BuildShare CLI installed successfully" -ForegroundColor Green
    Write-Host ""
    Write-Host "Version: $versionOutput"
    Write-Host ""
    Write-Host "Run:"
    Write-Host ""
    Write-Host "  buildshare --help"
    Write-Host ""

    if (-not $inPath) {
        Write-Host "Note: '$installDir' was added to your User PATH." -ForegroundColor Yellow
        Write-Host "If running in an external terminal window, restart PowerShell to refresh PATH." -ForegroundColor Yellow
        Write-Host ""
    }

} catch {
    Write-Host ""
    Write-Error "Installation failed: $($_.Exception.Message)"
    exit 1
} finally {
    if (Test-Path $tempDir) {
        Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}
