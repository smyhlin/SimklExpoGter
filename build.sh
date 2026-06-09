#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

APP_NAME="SimklExpoGter"
DIST_DIR="$ROOT_DIR/dist"
VERSION="${VERSION:-0.1.0}"

load_webkit_tag() {
  if [ -f .build/webkit.env ]; then
    # shellcheck disable=SC1091
    source .build/webkit.env
  fi
  : "${WAILS_WEBKIT_TAG:=}"
}

wails_cmd() {
  if command -v wails >/dev/null 2>&1; then
    wails "$@"
  else
    "$(go env GOPATH)/bin/wails" "$@"
  fi
}

ensure_frontend_dist() {
  mkdir -p frontend/dist
  if [ ! -f frontend/dist/.gitkeep ]; then
    : > frontend/dist/.gitkeep
  fi
}

frontend_build() {
  (cd frontend && npm ci && npm run build)
}

check() {
  bash -n build.sh scripts/bootstrap.sh
  if command -v shellcheck >/dev/null 2>&1; then
    shellcheck build.sh scripts/bootstrap.sh || true
  fi
  (cd frontend && npm run check)
  go test ./...
}

build_linux_gui() {
  load_webkit_tag
  frontend_build
  mkdir -p "$DIST_DIR/linux"
  local args=(build -clean -platform linux/amd64)
  if [ -n "$WAILS_WEBKIT_TAG" ]; then
    args+=(-tags "$WAILS_WEBKIT_TAG")
  fi
  wails_cmd "${args[@]}"
  cp -f "build/bin/$APP_NAME" "$DIST_DIR/linux/$APP_NAME"
}

build_linux_cli() {
  ensure_frontend_dist
  mkdir -p "$DIST_DIR/linux"
  go build -tags cli -trimpath -ldflags "-s -w" -o "$DIST_DIR/linux/${APP_NAME}-cli" ./
}

build_windows_cli() {
  ensure_frontend_dist
  mkdir -p "$DIST_DIR/windows"
  GOOS=windows GOARCH=amd64 go build -tags cli -trimpath -ldflags "-s -w" -o "$DIST_DIR/windows/${APP_NAME}-cli.exe" ./
}

build_deb() {
  build_linux_gui
  local pkgroot="$DIST_DIR/deb/${APP_NAME}_${VERSION}_amd64"
  rm -rf "$pkgroot"
  mkdir -p "$pkgroot/DEBIAN" "$pkgroot/usr/bin" "$pkgroot/usr/share/applications" "$pkgroot/usr/share/icons/hicolor/256x256/apps"
  cp "$DIST_DIR/linux/$APP_NAME" "$pkgroot/usr/bin/$APP_NAME"
  cp packaging/linux/${APP_NAME}.desktop "$pkgroot/usr/share/applications/${APP_NAME}.desktop"
  cp build/appicon.png "$pkgroot/usr/share/icons/hicolor/256x256/apps/${APP_NAME}.png"
  sed "s/@VERSION@/$VERSION/g" packaging/debian/control > "$pkgroot/DEBIAN/control"
  dpkg-deb --build "$pkgroot" "$DIST_DIR/${APP_NAME}_${VERSION}_amd64.deb"
}

build_appimage() {
  build_linux_gui
  local appdir="$DIST_DIR/appimage/${APP_NAME}.AppDir"
  rm -rf "$appdir"
  mkdir -p "$appdir/usr/bin" "$appdir/usr/share/applications" "$appdir/usr/share/icons/hicolor/256x256/apps"
  cp "$DIST_DIR/linux/$APP_NAME" "$appdir/usr/bin/$APP_NAME"
  cp packaging/appimage/AppRun "$appdir/AppRun"
  chmod +x "$appdir/AppRun"
  cp packaging/linux/${APP_NAME}.desktop "$appdir/${APP_NAME}.desktop"
  cp packaging/linux/${APP_NAME}.desktop "$appdir/usr/share/applications/${APP_NAME}.desktop"
  cp build/appicon.png "$appdir/${APP_NAME}.png"
  cp build/appicon.png "$appdir/usr/share/icons/hicolor/256x256/apps/${APP_NAME}.png"

  if [ -x ./linuxdeploy-x86_64.AppImage ]; then
    ./linuxdeploy-x86_64.AppImage --appdir "$appdir" --output appimage
  elif command -v linuxdeploy >/dev/null 2>&1; then
    linuxdeploy --appdir "$appdir" --output appimage
  else
    echo "linuxdeploy not found. AppDir prepared at $appdir"
    echo "Download linuxdeploy-x86_64.AppImage and rerun: ./build.sh appimage"
  fi
}

build_arch_package() {
  build_linux_gui
  local pkgroot="$DIST_DIR/arch/pkg"
  local package="$DIST_DIR/${APP_NAME}-${VERSION}-1-x86_64.pkg.tar.zst"
  rm -rf "$pkgroot"
  mkdir -p "$pkgroot/usr/bin" "$pkgroot/usr/share/applications" "$pkgroot/usr/share/icons/hicolor/256x256/apps" "$pkgroot/usr/share/licenses/simklexpogter"
  install -Dm755 "$DIST_DIR/linux/$APP_NAME" "$pkgroot/usr/bin/$APP_NAME"
  install -Dm644 packaging/linux/${APP_NAME}.desktop "$pkgroot/usr/share/applications/${APP_NAME}.desktop"
  install -Dm644 build/appicon.png "$pkgroot/usr/share/icons/hicolor/256x256/apps/${APP_NAME}.png"
  install -Dm644 LICENSE "$pkgroot/usr/share/licenses/simklexpogter/LICENSE"
  cat > "$pkgroot/.PKGINFO" <<PKGINFO
pkgname = simklexpogter
pkgbase = simklexpogter
pkgver = ${VERSION}-1
pkgdesc = Simkl backup/export desktop app with GUI, CLI and TUI modes
url = https://github.com/your-user/SimklExpoGter
builddate = $(date +%s)
packager = SimklExpoGter build.sh
size = $(du -sb "$pkgroot" | awk '{print $1}')
arch = x86_64
license = MIT
depend = gtk3
depend = webkit2gtk-4.1
depend = ca-certificates
PKGINFO
  if command -v bsdtar >/dev/null 2>&1; then
    (cd "$pkgroot" && bsdtar --zstd -cf "$package" .)
    echo "Arch package written to $package"
  else
    cp packaging/arch/PKGBUILD "$DIST_DIR/arch/PKGBUILD"
    echo "bsdtar not found. Package tree prepared at $pkgroot"
    echo "Reference PKGBUILD copied to $DIST_DIR/arch/PKGBUILD"
  fi
}

clean() {
  rm -rf "$DIST_DIR" build/bin frontend/dist .build
}

usage() {
  cat <<HELP
Usage: ./build.sh <command>

Commands:
  bootstrap     Run Linux bootstrap
  check         Run shell, frontend and Go checks
  frontend      Build frontend only
  linux         Build Linux GUI binary with Wails
  linux-cli     Build Linux CLI/TUI-only binary, no GUI/WebKit needed
  windows-cli   Cross-build Windows CLI/TUI-only binary
  deb           Build Debian package
  appimage      Build AppDir/AppImage package
  arch          Build Arch package or prepare package tree
  clean         Remove generated build outputs
HELP
}

case "${1:-help}" in
  bootstrap) ./scripts/bootstrap.sh "${@:2}" ;;
  check|test) check ;;
  frontend) frontend_build ;;
  linux|linux-gui) build_linux_gui ;;
  linux-cli) build_linux_cli ;;
  windows-cli) build_windows_cli ;;
  deb) build_deb ;;
  appimage) build_appimage ;;
  arch) build_arch_package ;;
  clean) clean ;;
  help|-h|--help) usage ;;
  *) echo "Unknown command: $1" >&2; usage; exit 2 ;;
esac
