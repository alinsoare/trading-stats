# Trading stats — Ubuntu / Debian `.deb` package

Produces a **self-contained** `.deb` that installs the native Go desktop app on
Ubuntu 22.04+ / Debian 12+.

Unlike the old Python build, nothing is downloaded at install time: the package
contains a single statically-linked Go binary. The only runtime requirements are
the standard desktop GL/X11 libraries (declared as `Depends`), which are already
present on any graphical Linux install.

## Build requirements (build machine)

```bash
sudo apt install golang-go gcc pkg-config libgl1-mesa-dev xorg-dev dpkg fakeroot
```

Go 1.23 or newer is required (install from [go.dev](https://go.dev/dl/) if the
distro package is older).

## Build

From anywhere in the repo:

```bash
linux/build_deb.sh
```

Output: `linux/dist/trading-stats_<version>_amd64.deb`

To build with a specific version:

```bash
PKG_VERSION=1.0.1 bash linux/build_deb.sh
```

Without `PKG_VERSION` the version defaults to `0.0.1` (local dev build). On CI the
workflow sets it from the git tag automatically.

## Install on target machine

```bash
sudo dpkg -i trading-stats_<version>_amd64.deb
sudo apt-get install -f          # resolves the GL/X11 runtime libs if missing
```

No internet access is required during installation.

## Run

```bash
trading-stats                    # binary at /usr/bin/trading-stats
```

Or find **Trading Stats** in the application menu (Finance category).

## What gets installed

| Path | Contents |
|------|----------|
| `/usr/bin/trading-stats` | The self-contained Go binary |
| `/usr/share/applications/trading-stats.desktop` | App-menu entry |
| `/usr/share/doc/trading-stats/copyright` | Build info |

No system Python packages, venvs, or pip installs are involved.

## Uninstall

```bash
sudo dpkg -r trading-stats       # remove
sudo dpkg -P trading-stats       # purge
```

## Notes

- Build must happen on a **Linux amd64** machine (the binary is native).
- For arm64, change `ARCH="amd64"` to `ARCH="arm64"` in `build_deb.sh` and build on that architecture.
- The `.deb` is **not** signed; Ubuntu may warn on install from outside a repository. Safe to proceed.
