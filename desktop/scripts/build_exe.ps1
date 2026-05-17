$ErrorActionPreference = "Stop"
$Root = Split-Path $PSScriptRoot -Parent
Set-Location $Root

if (-not (Get-Command pyinstaller -ErrorAction SilentlyContinue)) {
    Write-Error "PyInstaller not found. Run: pip install -e `".[build]`"  (and pip install -e ..)"
}

pyinstaller --clean --noconfirm (Join-Path $Root "TradingStats.spec")
Write-Host "Built: $(Join-Path $Root 'dist\TradingStats.exe')"
