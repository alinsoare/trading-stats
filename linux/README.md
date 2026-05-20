# Trading stats — Ubuntu / Debian `.deb` package

Produces a lightweight `.deb` that installs the PySide6 desktop app on Ubuntu 22.04+ / Debian 12+.

Third-party Python dependencies (PySide6, polars, matplotlib) are **not** embedded in the package. They are downloaded from PyPI and installed into a Python venv on the target machine at install time.

## Build requirements (build machine)

```bash
sudo apt install python3 python3-pip dpkg fakeroot
```

Python 3.10 or newer must be the default `python3`.

## Build

From anywhere in the repo:

```bash
trading-stats/linux/build_deb.sh
```

Output: `trading-stats/linux/dist/trading-stats_0.2.0_amd64.deb`

Build time is fast — only two small local wheels are built (no `pip install PySide6` at build time).

## Install on target machine

> **Internet access required** — `postinst` pip-installs PySide6, polars, and matplotlib from PyPI.

```bash
sudo dpkg -i trading-stats_0.2.0_amd64.deb
sudo apt-get install -f          # resolves python3-venv, python3-pip, libgl1 etc.
```

After `apt-get install -f` completes the install, `postinst` will automatically:
1. Create a Python venv at `/opt/trading-stats/venv`
2. `pip install` PySide6, polars, matplotlib and the two app wheels from PyPI

## Run

```bash
trading-stats                    # CLI launcher at /usr/local/bin/trading-stats
```

Or find **Trading Stats** in the application menu (Finance category).

## What gets installed

| Path | Contents |
|------|----------|
| `/opt/trading-stats/wheels/` | Local app wheels (`trading_stats`, `trading_stats_desktop`) — baked into the deb |
| `/opt/trading-stats/venv/` | Python venv created by `postinst` at install time — not in the deb |
| `/usr/local/bin/trading-stats` | Shell launcher |
| `/usr/share/applications/trading-stats.desktop` | App-menu entry |
| `/usr/share/doc/trading-stats/copyright` | Build info |

No system Python packages are touched.

## Uninstall

```bash
sudo dpkg -r trading-stats       # removes package files, keeps /opt/trading-stats (venv + wheels)
sudo dpkg -P trading-stats       # full purge including /opt/trading-stats
```

## Notes

- Build must happen on a **Linux amd64** machine (the wheels contain native Linux binaries).
- For arm64 / arm machines, change `ARCH="amd64"` to `ARCH="arm64"` in `build_deb.sh` and rebuild on that architecture.
- The `.deb` is **not** signed; Ubuntu may warn on install from outside a repository. Safe to proceed.
- If the target machine has no internet access, pre-download the required wheels and place them in `/opt/trading-stats/wheels/` before running `dpkg -i`, then modify `postinst` to use `--no-index --find-links`.
