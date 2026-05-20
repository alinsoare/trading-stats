# Trading stats — desktop (Windows / Linux)

A **normal desktop program** (PySide6) with the same analytics as the Streamlit app, reusing the
**`trading_stats`** package from the parent `trading-stats/` folder (no edits required there).

## Install (development)

From this directory (`trading-stats/desktop/`), with Python 3.10+:

```bash
pip install -e ..          # installs trading_stats core from trading-stats/
pip install -e .           # installs trading_stats_desktop (PySide6 UI)
```

Run:

```bash
python -m trading_stats_desktop
```

Or: `trading-stats-desktop` if the console script is on your `PATH`.

## Build a Windows installer

> **Target machine requirements:**
> - Python 3.10+ installed from [python.org](https://www.python.org/downloads/) with "Add Python to PATH" checked
> - Internet access (pip downloads PySide6, polars, matplotlib from PyPI at install time)

### Build requirements (build machine)

```powershell
choco install innosetup   # or download from https://jrsoftware.org/isinfo.php
```

Python 3.10+ must be on PATH.

### Build

From the repo root or `desktop\` folder:

```powershell
desktop\scripts\build_installer.ps1
```

Output: `desktop\dist\TradingStats-<version>-setup.exe` (~1–2 MB).

Build time is fast — only two small local wheels are built (no PySide6 download at build time).

### What the installer does

1. Copies the two local app wheels to `%LOCALAPPDATA%\TradingStats\wheels\`
2. Creates a Python venv at `%LOCALAPPDATA%\TradingStats\venv\`
3. pip-installs PySide6, polars, matplotlib and the app wheels from PyPI
4. Adds a **Trading Stats** shortcut to the Start Menu

No admin / UAC prompt required (user-level install).

> **Windows SmartScreen warning:** The installer is not code-signed. On first launch Windows may show
> "Windows protected your PC". Click **More info** → **Run anyway** to proceed.
> This is a one-time prompt per installer version.

### Uninstall

Use **Add or Remove Programs** → **Trading Stats**, or run the uninstaller from the Start Menu group. The entire `%LOCALAPPDATA%\TradingStats\` directory (venv + wheels) is removed.

## What it does

- **Data folders**: native folder picker + list of paths (saved in `QSettings` under your user profile).
- **One file per account**: the MQL5 script always overwrites `deals_<login>.csv`; no duplicates to manage.
- **Filters**: exit date range, accounts, rollup bucket.
- **KPIs + equity chart** (matplotlib) and **tables** for rollup, symbol, flows sample, per-account summary.
- **Export** filtered positions to CSV via file dialog.

Times are **broker/server time** from the CSV export, same as the web dashboard.
