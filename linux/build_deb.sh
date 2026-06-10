#!/usr/bin/env bash
# Build a self-contained .deb for Ubuntu / Debian (amd64).
#
# Unlike the old Python build, this packages a single statically-linked Go
# binary. No venv, no pip, no internet needed on the target machine — only the
# standard desktop GL/X11 runtime libraries (declared as Depends).
#
# Usage (from the repo root or this folder):
#   linux/build_deb.sh
#   PKG_VERSION=1.0.0 bash linux/build_deb.sh
#
# Build requirements (build machine):
#   - Go 1.23+
#   - gcc, pkg-config, libgl1-mesa-dev, xorg-dev   (Fyne CGO build deps)
#   - dpkg-deb, fakeroot
#
# Output:  linux/dist/trading-stats_<version>_amd64.deb
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
GO_DIR="$REPO_ROOT/go"

PKG_NAME="trading-stats"
PKG_VERSION="${PKG_VERSION:-0.0.1}"
ARCH="amd64"
DEB_FILE="${PKG_NAME}_${PKG_VERSION}_${ARCH}.deb"

STAGING="$SCRIPT_DIR/_staging"
BIN_DIR="$STAGING/usr/bin"
APPS_DIR="$STAGING/usr/share/applications"
DOC_DIR="$STAGING/usr/share/doc/$PKG_NAME"
DIST_DIR="$SCRIPT_DIR/dist"

echo "Version: $PKG_VERSION"

command -v go       >/dev/null || { echo "ERROR: go not found"; exit 1; }
command -v dpkg-deb >/dev/null || { echo "ERROR: dpkg-deb not found — run: sudo apt install dpkg"; exit 1; }
command -v fakeroot >/dev/null || { echo "ERROR: fakeroot not found — run: sudo apt install fakeroot"; exit 1; }

rm -rf "$STAGING"
mkdir -p "$BIN_DIR" "$APPS_DIR" "$DOC_DIR" "$STAGING/DEBIAN" "$DIST_DIR"

echo "Building Go binary..."
( cd "$GO_DIR" && CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o "$BIN_DIR/trading-stats" . )
chmod 755 "$BIN_DIR/trading-stats"

cat > "$APPS_DIR/trading-stats.desktop" << 'DESKTOP'
[Desktop Entry]
Name=Trading Stats
GenericName=MT5 Trading Statistics
Comment=Analyse MetaTrader 5 deal CSV exports across multiple accounts
Exec=/usr/bin/trading-stats
Terminal=false
Type=Application
Categories=Finance;Office;
Keywords=trading;forex;mt5;metatrader;statistics;
DESKTOP

cat > "$DOC_DIR/copyright" << EOF
trading-stats $PKG_VERSION
Source: local build from $(realpath "$REPO_ROOT")
Build date: $(date -u +"%Y-%m-%d %H:%M UTC")
EOF

INSTALLED_KB=$(du -sk "$BIN_DIR" | awk '{print $1}')

cat > "$STAGING/DEBIAN/control" << EOF
Package: $PKG_NAME
Version: $PKG_VERSION
Architecture: $ARCH
Maintainer: local build
Installed-Size: $INSTALLED_KB
Depends: libc6, libgl1, libx11-6, libxcursor1, libxrandr2, libxinerama1, libxi6, libxxf86vm1
Section: finance
Priority: optional
Description: MT5 multi-account trading statistics — native desktop app
 Analyses MetaTrader 5 deal CSV exports across multiple accounts.
 Provides KPIs (win rate, profit factor, drawdown, equity curve) with
 date / account filters and CSV export. Ships as a single self-contained
 Go binary — no Python, no pip, no internet required at install time.
EOF

cat > "$STAGING/DEBIAN/postinst" << 'POSTINST'
#!/bin/sh
set -e
if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database /usr/share/applications || true
fi
POSTINST
chmod 755 "$STAGING/DEBIAN/postinst"

cat > "$STAGING/DEBIAN/postrm" << 'POSTRM'
#!/bin/sh
set -e
if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database /usr/share/applications || true
fi
POSTRM
chmod 755 "$STAGING/DEBIAN/postrm"

find "$STAGING" -type d -exec chmod 755 {} \;
find "$STAGING/usr" -type f -exec chmod 644 {} \;
chmod 755 "$BIN_DIR/trading-stats"
chmod 755 "$STAGING/DEBIAN/postinst" "$STAGING/DEBIAN/postrm"

echo "Building $DEB_FILE..."
fakeroot dpkg-deb --build --root-owner-group "$STAGING" "$DIST_DIR/$DEB_FILE"

echo ""
echo "Done: $DIST_DIR/$DEB_FILE"
echo "Install with: sudo dpkg -i $DIST_DIR/$DEB_FILE"
