# adenVault Windows installer
# Usage: irm https://raw.githubusercontent.com/your-org/aden-vault/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$repo    = "codebyNJ/AdenVault"
$binary  = "adenV.exe"
$installDir = "$env:USERPROFILE\.adenV\bin"

Write-Host ""
Write-Host "  adenVault installer" -ForegroundColor Magenta
Write-Host "  a vault that lives in your home dir, not the cloud" -ForegroundColor DarkGray
Write-Host ""

# ── resolve the latest release tag ───────────────────────────────────────────
Write-Host "  fetching latest release..." -ForegroundColor DarkGray

$release = Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest"
$tag     = $release.tag_name

Write-Host "  latest: $tag" -ForegroundColor Cyan

# ── pick the right asset ─────────────────────────────────────────────────────
$arch   = if ([System.Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
$asset  = $release.assets | Where-Object { $_.name -like "*windows*$arch*" -or $_.name -eq $binary }

if (-not $asset) {
    # fallback: look for a plain adenV.exe asset
    $asset = $release.assets | Where-Object { $_.name -eq $binary }
}

if (-not $asset) {
    Write-Host ""
    Write-Host "  no Windows binary found in release $tag." -ForegroundColor Red
    Write-Host "  build from source: https://github.com/$repo#how-to-install" -ForegroundColor DarkGray
    exit 1
}

$downloadUrl = $asset.browser_download_url

# ── download ─────────────────────────────────────────────────────────────────
Write-Host "  downloading $($asset.name)..." -ForegroundColor DarkGray

New-Item -ItemType Directory -Force $installDir | Out-Null
$dest = Join-Path $installDir $binary
Invoke-WebRequest -Uri $downloadUrl -OutFile $dest -UseBasicParsing

# ── PATH registration (current user, permanent) ───────────────────────────────
$userPath = [System.Environment]::GetEnvironmentVariable("PATH", "User")
if ($userPath -notlike "*$installDir*") {
    [System.Environment]::SetEnvironmentVariable(
        "PATH",
        "$installDir;$userPath",
        "User"
    )
    Write-Host "  added $installDir to your PATH" -ForegroundColor DarkGray
} else {
    Write-Host "  $installDir is already on your PATH" -ForegroundColor DarkGray
}

# also update the current session
$env:PATH = "$installDir;$env:PATH"

# ── verify ───────────────────────────────────────────────────────────────────
Write-Host ""
& $dest --version

Write-Host ""
Write-Host "  adenV is ready." -ForegroundColor Green
Write-Host "  restart your terminal, then run: adenV init" -ForegroundColor DarkGray
Write-Host ""
