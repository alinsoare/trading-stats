# Trading stats — macOS `.dmg`

Produces a **self-contained** `.dmg` containing a universal `TradingStats.app`
(Intel `x86_64` + Apple Silicon `arm64`, merged with `lipo`).

Like the Linux and Windows builds, nothing is downloaded at install time: the
`.app` bundles a single universal Go binary. No Python, no runtime, no internet
needed on the target machine.

## Build requirements (build machine — must be macOS)

- Go 1.23 or newer (install from [go.dev](https://go.dev/dl/))
- Xcode Command Line Tools (provides `clang`, `lipo`, `hdiutil`, `sips`, `iconutil`):

```bash
xcode-select --install
```

## Build

From anywhere in the repo:

```bash
macos/build_app.sh
```

Output: `macos/dist/TradingStats-<version>.dmg`

To build with a specific version:

```bash
PKG_VERSION=1.0.1 bash macos/build_app.sh
```

Without `PKG_VERSION` the version defaults to `0.0.1` (local dev build). On CI the
workflow sets it from the git tag automatically.

## Install on target machine

1. Open `TradingStats-<version>.dmg`.
2. Drag **Trading Stats** onto the **Applications** shortcut.
3. Launch it from Launchpad or `/Applications`.

## Run

Open **Trading Stats** from Launchpad / Applications, or from a terminal:

```bash
open -a "Trading Stats"
```

## Gatekeeper (unsigned app)

The `.app` is **not** code-signed or notarized. On first launch macOS may refuse
to open it ("Apple could not verify…" or "is damaged"). Work around it with one
of:

- **Right-click** the app in `/Applications` → **Open** → **Open**, or
- clear the quarantine attribute:

```bash
xattr -dr com.apple.quarantine "/Applications/TradingStats.app"
```

## Notes

- Build must run on **macOS** (`lipo`, `hdiutil`, `sips`, `iconutil` are macOS-only).
- The universal binary runs natively on both Intel and Apple Silicon Macs.
- The app icon is generated from `mql5/ExportTradingDeals_logo_200x200.png`.
