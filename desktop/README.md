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

## Build a single `.exe` (Windows)

On a Windows machine:

```powershell
cd trading-stats\desktop
pip install -e ..
pip install -e ".[build]"
.\scripts\build_exe.ps1
```

Output: `dist\TradingStats.exe` (one file, no console window).

First launch may be slow (AV scanning). If Windows SmartScreen warns, use "More info" → "Run anyway"
or sign the binary later.

## What it does

- **Data folders**: native folder picker + list of paths (saved in `QSettings` under your user profile).
- **One file per account**: the MQL5 script always overwrites `deals_<login>.csv`; no duplicates to manage.
- **Filters**: exit date range, accounts, rollup bucket.
- **KPIs + equity chart** (matplotlib) and **tables** for rollup, symbol, flows sample, per-account summary.
- **Export** filtered positions to CSV via file dialog.

Times are **broker/server time** from the CSV export, same as the web dashboard.
