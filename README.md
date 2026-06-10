# Trading Stats

Multi-account MT5 deal CSV export + native desktop analytics app, written in Go.

| Component | Location | Purpose |
|-----------|----------|---------|
| **Desktop app** | `go/` | Native Go + Fyne UI (ingest, KPIs, filters, equity chart, tables) |
| **Ubuntu `.deb`** | `linux/` | Self-contained package builder |
| **MQL5 exporter** | `mql5/` | MetaEditor script that writes `deals_*.csv` |

The app ships as a **single self-contained binary**. There is no Python, no
runtime to install, and no internet needed at install time — unlike a typical
interpreted desktop app, the Go binary bundles everything it needs.

---

## Pre-built packages

Pre-compiled binaries are available on the
[Releases page](https://github.com/alinsoare/trading-stats/releases):

- `trading-stats_<version>_amd64.deb` — Ubuntu / Debian package
- `trading-stats-linux-amd64` — bare Linux binary (no install)
- `TradingStats-<version>.exe` — Windows executable (run directly, no install)

```bash
# Ubuntu / Debian
sudo dpkg -i trading-stats_<version>_amd64.deb
```

On Windows, just download `TradingStats-<version>.exe` and run it.

---

## Run from source

Requires **Go 1.23+**. On Linux you also need the Fyne build dependencies
(a C compiler and the X11/GL development headers):

```bash
# Ubuntu / Debian build deps (one-time)
sudo apt install gcc pkg-config libgl1-mesa-dev xorg-dev

cd go
go run .
```

On Windows, install Go and a C compiler (the MinGW `gcc` toolchain), then:

```powershell
cd go
go run .
```

Run the tests (KPI parity fixtures are in `go/testdata/parity`):

```bash
cd go
go test ./...
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

Paths and per-account breakeven tolerances are saved automatically (via Fyne
preferences, under your user config dir) and restored on next launch.

---

## Ubuntu `.deb` package (build from source)

See `linux/README.md`. One-command build:

```bash
linux/build_deb.sh
sudo dpkg -i linux/dist/trading-stats_<version>_amd64.deb
```

---

## Windows `.exe` (build from source)

With Go and a C compiler on `PATH`:

```powershell
cd go
go build -ldflags "-s -w -H=windowsgui" -o TradingStats.exe .
```

The resulting `.exe` is self-contained and runs without any install step.

> **Windows SmartScreen:** the binary is not code-signed. On first launch
> Windows may show "Windows protected your PC" — click **More info** →
> **Run anyway**.
