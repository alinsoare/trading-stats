#!/usr/bin/env bash
# Build a self-contained macOS .dmg containing a universal TradingStats.app.
#
# The .app bundles a single universal Go binary (Intel x86_64 + Apple Silicon
# arm64, merged with lipo). No Python, no runtime, no internet needed on the
# target machine.
#
# Usage (from the repo root or this folder):
#   macos/build_app.sh
#   PKG_VERSION=1.0.0 bash macos/build_app.sh
#
# Build requirements (build machine — must be macOS):
#   - Go 1.23+
#   - Xcode Command Line Tools (clang, lipo, hdiutil, sips, iconutil)
#
# Output:  macos/dist/TradingStats-<version>.dmg
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
GO_DIR="$REPO_ROOT/go"

APP_NAME="TradingStats"
BIN_NAME="trading-stats"
BUNDLE_ID="com.alinsoare.tradingstats"
PKG_VERSION="${PKG_VERSION:-0.0.1}"
ICON_SRC="$REPO_ROOT/mql5/ExportTradingDeals_logo_200x200.png"

BUILD="$SCRIPT_DIR/_build"
APP="$BUILD/$APP_NAME.app"
DMGROOT="$BUILD/dmgroot"
DIST_DIR="$SCRIPT_DIR/dist"
DMG_FILE="$DIST_DIR/${APP_NAME}-${PKG_VERSION}.dmg"

echo "Version: $PKG_VERSION"

[ "$(uname -s)" = "Darwin" ] || { echo "ERROR: must build on macOS"; exit 1; }
command -v go       >/dev/null || { echo "ERROR: go not found"; exit 1; }
command -v lipo     >/dev/null || { echo "ERROR: lipo not found — install Xcode Command Line Tools"; exit 1; }
command -v hdiutil  >/dev/null || { echo "ERROR: hdiutil not found"; exit 1; }
command -v sips     >/dev/null || { echo "ERROR: sips not found"; exit 1; }
command -v iconutil >/dev/null || { echo "ERROR: iconutil not found"; exit 1; }

rm -rf "$BUILD"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources" "$DIST_DIR"

echo "Building arm64 binary..."
( cd "$GO_DIR" && CGO_ENABLED=1 GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o "$BUILD/ts-arm64" . )

echo "Building amd64 binary..."
( cd "$GO_DIR" && CGO_ENABLED=1 GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o "$BUILD/ts-amd64" . )

echo "Merging into universal binary..."
lipo -create -output "$APP/Contents/MacOS/$BIN_NAME" "$BUILD/ts-arm64" "$BUILD/ts-amd64"
chmod 755 "$APP/Contents/MacOS/$BIN_NAME"

echo "Generating icon..."
ICONSET="$BUILD/icon.iconset"
mkdir -p "$ICONSET"
for size in 16 32 64 128 256 512; do
    sips -z "$size" "$size"        "$ICON_SRC" --out "$ICONSET/icon_${size}x${size}.png"      >/dev/null
    sips -z "$((size*2))" "$((size*2))" "$ICON_SRC" --out "$ICONSET/icon_${size}x${size}@2x.png" >/dev/null
done
iconutil -c icns "$ICONSET" -o "$APP/Contents/Resources/icon.icns"

echo "Writing Info.plist..."
cat > "$APP/Contents/Info.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>$BIN_NAME</string>
    <key>CFBundleIdentifier</key>
    <string>$BUNDLE_ID</string>
    <key>CFBundleName</key>
    <string>Trading Stats</string>
    <key>CFBundleDisplayName</key>
    <string>Trading Stats</string>
    <key>CFBundleIconFile</key>
    <string>icon.icns</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>$PKG_VERSION</string>
    <key>CFBundleVersion</key>
    <string>$PKG_VERSION</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>LSMinimumSystemVersion</key>
    <string>11.0</string>
    <key>LSApplicationCategoryType</key>
    <string>public.app-category.finance</string>
</dict>
</plist>
EOF

echo "Building $DMG_FILE..."
rm -rf "$DMGROOT"
mkdir -p "$DMGROOT"
cp -R "$APP" "$DMGROOT/"
ln -s /Applications "$DMGROOT/Applications"
rm -f "$DMG_FILE"
hdiutil create -volname "Trading Stats" -srcfolder "$DMGROOT" -ov -format UDZO "$DMG_FILE"

echo ""
echo "Done: $DMG_FILE"
echo "Open the .dmg and drag Trading Stats to Applications."
