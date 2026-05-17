#!/usr/bin/env bash
# Build a self-contained .deb for Ubuntu / Debian (amd64).
#
# Usage (from the repo root or this folder):
#   trading-stats/linux/build_deb.sh
#
# Requirements on the build machine:
#   - Python 3.10+    (python3)
#   - dpkg-deb        (apt install dpkg)
#   - fakeroot        (apt install fakeroot)   — needed for correct ownership
#
# Output:  trading-stats/linux/dist/trading-stats_<version>_amd64.deb
#
# What gets installed on the target machine:
#   /opt/trading-stats/venv/          embedded Python venv (Polars, PySide6, matplotlib)
#   /opt/trading-stats/               (nothing else; source is baked into the venv via pip)
#   /usr/local/bin/trading-stats      launcher script
#   /usr/share/applications/trading-stats.desktop
#   /usr/share/doc/trading-stats/copyright
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

PKG_NAME="trading-stats"
PKG_VERSION="0.1.0"
ARCH="amd64"
DEB_FILE="${PKG_NAME}_${PKG_VERSION}_${ARCH}.deb"

STAGING="$SCRIPT_DIR/_staging"
OPT_DIR="$STAGING/opt/trading-stats"
BIN_DIR="$STAGING/usr/local/bin"
APPS_DIR="$STAGING/usr/share/applications"
DOC_DIR="$STAGING/usr/share/doc/$PKG_NAME"
DIST_DIR="$SCRIPT_DIR/dist"

# ── sanity checks ────────────────────────────────────────────────────────────
command -v python3  >/dev/null || { echo "ERROR: python3 not found"; exit 1; }
command -v dpkg-deb >/dev/null || { echo "ERROR: dpkg-deb not found — run: sudo apt install dpkg"; exit 1; }
command -v fakeroot >/dev/null || { echo "ERROR: fakeroot not found — run: sudo apt install fakeroot"; exit 1; }

PY_VER=$(python3 -c "import sys; print(f'{sys.version_info.major}.{sys.version_info.minor}')")
if python3 -c "import sys; sys.exit(0 if sys.version_info >= (3,10) else 1)"; then
    echo "Python $PY_VER — OK"
else
    echo "ERROR: Python 3.10+ required (found $PY_VER)"; exit 1
fi

# ── clean staging ────────────────────────────────────────────────────────────
echo "Cleaning staging dir..."
rm -rf "$STAGING"
mkdir -p "$OPT_DIR" "$BIN_DIR" "$APPS_DIR" "$DOC_DIR" "$STAGING/DEBIAN" "$DIST_DIR"

# ── build venv with copied binaries (portable; no symlinks to host python) ──
echo "Creating venv..."
python3 -m venv --copies "$OPT_DIR/venv"

echo "Installing dependencies..."
"$OPT_DIR/venv/bin/pip" install --upgrade pip --quiet

# Install core package (trading_stats)
"$OPT_DIR/venv/bin/pip" install "$REPO_ROOT" --quiet

# Install desktop package (trading_stats_desktop — PySide6 UI)
"$OPT_DIR/venv/bin/pip" install "$REPO_ROOT/desktop" --quiet

echo "Packages installed:"
"$OPT_DIR/venv/bin/pip" list --format=columns | grep -E "trading|polars|PySide6|matplotlib"

# ── launcher ─────────────────────────────────────────────────────────────────
cat > "$BIN_DIR/trading-stats" << 'LAUNCHER'
#!/usr/bin/env bash
exec /opt/trading-stats/venv/bin/python -m trading_stats_desktop "$@"
LAUNCHER
chmod 755 "$BIN_DIR/trading-stats"

# ── .desktop entry ───────────────────────────────────────────────────────────
cat > "$APPS_DIR/trading-stats.desktop" << 'DESKTOP'
[Desktop Entry]
Name=Trading Stats
GenericName=MT5 Trading Statistics
Comment=Analyse MetaTrader 5 deal CSV exports across multiple accounts
Exec=/opt/trading-stats/venv/bin/python -m trading_stats_desktop
Terminal=false
Type=Application
Categories=Finance;Office;
Keywords=trading;forex;mt5;metatrader;statistics;
DESKTOP

# ── copyright ─────────────────────────────────────────────────────────────────
cat > "$DOC_DIR/copyright" << EOF
trading-stats $PKG_VERSION
Source: local build from $(realpath "$REPO_ROOT")
Build date: $(date -u +"%Y-%m-%d %H:%M UTC")
EOF

# ── DEBIAN/control ───────────────────────────────────────────────────────────
INSTALLED_KB=$(du -sk "$OPT_DIR/venv" | awk '{print $1}')

cat > "$STAGING/DEBIAN/control" << EOF
Package: $PKG_NAME
Version: $PKG_VERSION
Architecture: $ARCH
Maintainer: local build
Installed-Size: $INSTALLED_KB
Depends: python3 (>= 3.10), libgl1, libglib2.0-0, libdbus-1-3
Section: finance
Priority: optional
Description: MT5 multi-account trading statistics — PySide6 desktop app
 Analyses MetaTrader 5 deal CSV exports across multiple accounts.
 Provides KPIs (win rate, profit factor, drawdown, equity curve) with
 date / account filters and CSV export. Bundles a Python venv with
 Polars, PySide6 and matplotlib — no system pip install required.
EOF

# ── DEBIAN/postinst ──────────────────────────────────────────────────────────
cat > "$STAGING/DEBIAN/postinst" << 'POSTINST'
#!/bin/sh
set -e
if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database /usr/share/applications || true
fi
POSTINST
chmod 755 "$STAGING/DEBIAN/postinst"

# ── DEBIAN/postrm ─────────────────────────────────────────────────────────────
cat > "$STAGING/DEBIAN/postrm" << 'POSTRM'
#!/bin/sh
set -e
if [ "$1" = "purge" ]; then
    rm -rf /opt/trading-stats
fi
if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database /usr/share/applications || true
fi
POSTRM
chmod 755 "$STAGING/DEBIAN/postrm"

# ── fix permissions ───────────────────────────────────────────────────────────
find "$STAGING" -type d  -exec chmod 755 {} \;
find "$STAGING" -type f  -exec chmod 644 {} \;
chmod 755 "$BIN_DIR/trading-stats"
chmod 755 "$STAGING/DEBIAN/postinst"
chmod 755 "$STAGING/DEBIAN/postrm"
# venv executables must stay executable
find "$OPT_DIR/venv/bin" -type f -exec chmod 755 {} \;

# ── build deb ─────────────────────────────────────────────────────────────────
echo "Building $DEB_FILE..."
fakeroot dpkg-deb --build --root-owner-group "$STAGING" "$DIST_DIR/$DEB_FILE"

echo ""
echo "Done: $DIST_DIR/$DEB_FILE"
echo ""
echo "Install with:"
echo "  sudo dpkg -i $DIST_DIR/$DEB_FILE"
echo "  sudo apt-get install -f   # if missing system libs (libgl1 etc.)"
echo ""
echo "Then run:  trading-stats"
echo "     or:   python3 -m trading_stats_desktop"
