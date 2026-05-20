# Trading Stats

Multi-account MT5 deal CSV export + PySide6 desktop analytics app.

| Component | Location | Purpose |
|-----------|----------|---------|
| **Core library** | `src/trading_stats/` | CSV ingest, KPI calculations, path resolution |
| **Desktop app** | `desktop/` | PySide6 native UI (filters, charts, tables) |
| **Ubuntu `.deb`** | `linux/` | Installable package builder |
| **MQL5 exporter** | `mql5/` | MetaEditor script that writes `deals_*.csv` |

---

## Pre-built packages

Pre-compiled Ubuntu `.deb` and Windows `.exe` packages are available on the
[Releases page](https://github.com/alinsoare/trading-stats/releases) — no build step required.

```bash
# Ubuntu / Debian
sudo dpkg -i trading-stats_<version>_amd64.deb
```

```powershell
# Windows — download TradingStats-<version>.exe and run directly
```

---

## Quick start (run from source)

```bash
python3 -m venv .venv
source .venv/bin/activate        # Windows: .venv\Scripts\activate
pip install -e .                 # core library
pip install -e desktop/          # desktop app (PySide6 + matplotlib)
python -m trading_stats_desktop  # launch
```

---

## MQL5 export

Compile `mql5/ExportTradingDeals.mq5` in MetaEditor, copy `.ex5` into each
terminal's `MQL5/Scripts/`, run it from the MT5 Navigator.
CSV files appear under `MQL5/Files/trading_stats/` (comma-separated, dot decimals, ANSI encoding).

---

## Configuration

In the desktop app, add **absolute paths** to the folders containing `deals_*.csv`:

- the folder itself (`…/MQL5/Files/trading_stats`), **or**
- the MT5 terminal root if `MQL5/Files/trading_stats` exists under it.

Paths are saved automatically in `~/.config/TradingStats/` and restored on next launch.

---

## Ubuntu `.deb` package (build from source)

See `linux/README.md`. One-command build:

```bash
linux/build_deb.sh
sudo dpkg -i linux/dist/trading-stats_<version>_amd64.deb
```

---

## Windows `.exe` (build from source)

```powershell
pip install -e .
pip install -e desktop\
python -m trading_stats_desktop
```

For a standalone `.exe` see `desktop/README.md`.
