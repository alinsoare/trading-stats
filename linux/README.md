# Trading stats — Ubuntu / Debian `.deb` package

Produces a self-contained `.deb` that installs the PySide6 desktop app on Ubuntu 22.04+ / Debian 12+.

## Build requirements (build machine)

```bash
sudo apt install python3 python3-venv dpkg fakeroot
```

Python 3.10 or newer must be the default `python3`.

## Build

From anywhere in the repo:

```bash
trading-stats/linux/build_deb.sh
```

Output: `trading-stats/linux/dist/trading-stats_0.1.0_amd64.deb`

Build time is dominated by `pip install PySide6` (~1–2 min first run).

## Install on target machine

```bash
sudo dpkg -i trading-stats_0.1.0_amd64.deb
sudo apt-get install -f          # resolves any missing system libs (libgl1 etc.)
```

## Run

```bash
trading-stats                    # CLI launcher at /usr/local/bin/trading-stats
```

Or find **Trading Stats** in the application menu (Finance category).

## What gets installed

| Path | Contents |
|------|----------|
| `/opt/trading-stats/venv/` | Embedded Python venv — Polars, PySide6, matplotlib, trading_stats, trading_stats_desktop |
| `/usr/local/bin/trading-stats` | Shell launcher |
| `/usr/share/applications/trading-stats.desktop` | App-menu entry |
| `/usr/share/doc/trading-stats/copyright` | Build info |

No system Python packages are touched.

## Uninstall

```bash
sudo dpkg -r trading-stats       # removes files, keeps /opt/trading-stats
sudo dpkg -P trading-stats       # full purge including /opt/trading-stats
```

## Notes

- Build must happen on a **Linux amd64** machine (the venv contains native Linux wheels).
- For arm64 / arm machines, change `ARCH="amd64"` to `ARCH="arm64"` in `build_deb.sh` and rebuild on that architecture.
- The `.deb` is **not** signed; Ubuntu may warn on install from outside a repository. Safe to proceed.
