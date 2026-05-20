#Requires -Version 5.1
# Build the Trading Stats Windows installer.
#
# Usage (from repo root or desktop\ folder):
#   desktop\scripts\build_installer.ps1
#
# Requirements:
#   - Python 3.10+    (python3 / python on PATH)
#   - Inno Setup 6+   (https://jrsoftware.org/isinfo.php  or  choco install innosetup)
#
# PKG_VERSION env var sets the installer version.
# Falls back to the current exact git tag (v-prefix stripped), then "0.1.0".
#
# Output: desktop\dist\TradingStats-<version>-setup.exe

$ErrorActionPreference = "Stop"

$ScriptDir  = Split-Path $MyInvocation.MyCommand.Path -Parent
$DesktopDir = Split-Path $ScriptDir -Parent
$RepoRoot   = Split-Path $DesktopDir -Parent

Set-Location $DesktopDir

# ── resolve version ───────────────────────────────────────────────────────────
$PkgVersion = $Env:PKG_VERSION
if (-not $PkgVersion) {
    try {
        $tag = git -C $RepoRoot describe --tags --exact-match 2>$null
        if ($tag) { $PkgVersion = $tag -replace '^v', '' }
    } catch {}
}
if (-not $PkgVersion) { $PkgVersion = "0.1.0" }
Write-Host "Version: $PkgVersion"

# ── sanity checks ─────────────────────────────────────────────────────────────
if (-not (Get-Command python -ErrorAction SilentlyContinue)) {
    Write-Error "python not found on PATH."
}
if (-not (Get-Command iscc -ErrorAction SilentlyContinue)) {
    Write-Error "iscc (Inno Setup Compiler) not found. Install with: choco install innosetup"
}

# ── clean wheel output dir ────────────────────────────────────────────────────
$WheelDir = Join-Path $DesktopDir "dist\wheels"
if (Test-Path $WheelDir) { Remove-Item $WheelDir -Recurse -Force }
New-Item -ItemType Directory -Path $WheelDir | Out-Null

# ── build local wheels (no third-party deps embedded) ─────────────────────────
Write-Host "Building wheel: trading_stats..."
python -m pip wheel $RepoRoot --no-deps -w $WheelDir --quiet

Write-Host "Building wheel: trading_stats_desktop..."
python -m pip wheel $DesktopDir --no-deps -w $WheelDir --quiet

Write-Host "Wheels built:"
Get-ChildItem $WheelDir | ForEach-Object { Write-Host "  $($_.Name)  ($([math]::Round($_.Length/1KB, 1)) KB)" }

# ── compile installer ─────────────────────────────────────────────────────────
Write-Host "Compiling installer (version $PkgVersion)..."
iscc "$DesktopDir\TradingStats.iss" /DAppVersion="$PkgVersion"

$InstallerPath = Join-Path $DesktopDir "dist\TradingStats-$PkgVersion-setup.exe"
if (Test-Path $InstallerPath) {
    $SizeMB = [math]::Round((Get-Item $InstallerPath).Length / 1MB, 2)
    Write-Host ""
    Write-Host "Done: $InstallerPath  ($SizeMB MB)"
} else {
    Write-Error "Installer not found at expected path: $InstallerPath"
}
