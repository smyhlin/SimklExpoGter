#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

APP_NAME="${APP_NAME:-SimklExpoGter}"
DIST_DIR="$ROOT_DIR/dist"
VERSION="${VERSION:-0.1.0}"
GOOS_HOST="$(go env GOOS 2>/dev/null || echo unknown)"
GOARCH_HOST="$(go env GOARCH 2>/dev/null || echo unknown)"

log() {
  printf '\033[1;36m==>\033[0m %s\n' "$*"
}

warn() {
  printf '\033[1;33mWARN:\033[0m %s\n' "$*" >&2
}

die() {
  printf '\033[1;31mERROR:\033[0m %s\n' "$*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1
}

wails_cmd() {
  if need_cmd wails; then
    wails "$@"
    return
  fi

  local gopath_bin
  gopath_bin="$(go env GOPATH)/bin/wails"
  if [ -x "$gopath_bin" ]; then
    "$gopath_bin" "$@"
    return
  fi

  die "Wails CLI not found. Run ./scripts/bootstrap.sh or go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0"
}

ensure_frontend_dist() {
  mkdir -p frontend/dist
  if [ ! -f frontend/dist/.gitkeep ]; then
    : > frontend/dist/.gitkeep
  fi
}

frontend_install() {
  log "Installing frontend dependencies"
  (cd frontend && npm ci)
}

frontend_build() {
  frontend_install
  log "Building frontend"
  (cd frontend && npm run build)
}

check() {
  log "Checking shell scripts"
  bash -n build.sh scripts/bootstrap.sh packaging/appimage/AppRun

  if need_cmd shellcheck; then
    shellcheck build.sh scripts/bootstrap.sh packaging/appimage/AppRun || true
  fi

  if [ ! -d frontend/node_modules ]; then
    frontend_install
  fi

  log "Checking frontend"
  (cd frontend && npm run check)

  log "Running Go tests"
  go test ./...
}

detect_webkit_tag() {
  # Explicit override wins. Support both names because earlier local docs/scripts used WAILS_TAGS.
  if [ -n "${WAILS_WEBKIT_TAG:-}" ]; then
    printf '%s' "$WAILS_WEBKIT_TAG"
    return
  fi
  if [ -n "${WAILS_TAGS:-}" ]; then
    printf '%s' "$WAILS_TAGS"
    return
  fi

  if [ -f .build/webkit.env ]; then
    # shellcheck disable=SC1091
    source .build/webkit.env
    if [ -n "${WAILS_WEBKIT_TAG:-}" ]; then
      printf '%s' "$WAILS_WEBKIT_TAG"
      return
    fi
    if [ -n "${WAILS_TAGS:-}" ]; then
      printf '%s' "$WAILS_TAGS"
      return
    fi
  fi

  if need_cmd pkg-config; then
    if pkg-config --exists webkit2gtk-4.1 2>/dev/null; then
      printf 'webkit2_41'
      return
    fi
    if pkg-config --exists webkit2gtk-4.0 2>/dev/null; then
      printf ''
      return
    fi
  fi

  # Safe fallback for modern distros; Wails v2 needs this when only WebKitGTK 4.1 exists.
  printf 'webkit2_41'
}

webkit_suffix() {
  local tag="${1:-}"
  case "$tag" in
    webkit2_41) printf 'webkitgtk41' ;;
    "") printf 'webkitgtk40' ;;
    *) printf '%s' "$tag" | tr -c '[:alnum:]' '-' ;;
  esac
}

wails_build_args_linux() {
  local tag
  tag="$(detect_webkit_tag)"
  local args=(build -clean -platform linux/amd64)
  if [ -n "$tag" ]; then
    args+=(-tags "$tag")
  fi
  printf '%s\0' "${args[@]}"
}

build_linux_gui() {
  [ "$GOOS_HOST" = "linux" ] || die "Linux GUI build must run on Linux."

  local tag
  tag="$(detect_webkit_tag)"
  log "Building Linux GUI with Wails${tag:+, tags: $tag}"

  frontend_build
  mkdir -p "$DIST_DIR/linux"

  local args=()
  while IFS= read -r -d '' arg; do
    args+=("$arg")
  done < <(wails_build_args_linux)

  wails_cmd "${args[@]}"

  install -Dm755 "build/bin/$APP_NAME" "$DIST_DIR/linux/$APP_NAME"
}

build_linux_cli() {
  log "Building Linux CLI/TUI"
  ensure_frontend_dist
  mkdir -p "$DIST_DIR/linux"
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go build -tags cli -trimpath -ldflags "-s -w" -o "$DIST_DIR/linux/${APP_NAME}-cli" ./
  chmod +x "$DIST_DIR/linux/${APP_NAME}-cli"
}

build_windows_gui() {
  log "Building Windows GUI with Wails"
  frontend_build
  mkdir -p "$DIST_DIR/windows"

  # Wails v2 Windows builds are Go/WebView2 based and can be targeted from
  # non-Windows hosts. The produced .exe still must be tested on Windows.
  wails_cmd build -clean -platform windows/amd64

  install -Dm755 "build/bin/${APP_NAME}.exe" "$DIST_DIR/windows/${APP_NAME}.exe"
}

build_windows_cli() {
  log "Cross-building Windows CLI/TUI"
  ensure_frontend_dist
  mkdir -p "$DIST_DIR/windows"
  GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
    go build -tags cli -trimpath -ldflags "-s -w" -o "$DIST_DIR/windows/${APP_NAME}-cli.exe" ./
}

build_deb() {
  [ "$GOOS_HOST" = "linux" ] || die "Debian package build must run on Linux."
  need_cmd dpkg-deb || die "dpkg-deb not found."

  build_linux_gui

  local pkgroot="$DIST_DIR/deb/${APP_NAME}_${VERSION}_amd64"
  rm -rf "$pkgroot"
  mkdir -p "$pkgroot/DEBIAN" "$pkgroot/usr/bin" "$pkgroot/usr/share/applications" "$pkgroot/usr/share/icons/hicolor/256x256/apps"

  install -Dm755 "$DIST_DIR/linux/$APP_NAME" "$pkgroot/usr/bin/$APP_NAME"
  install -Dm644 packaging/linux/${APP_NAME}.desktop "$pkgroot/usr/share/applications/${APP_NAME}.desktop"
  install -Dm644 build/appicon.png "$pkgroot/usr/share/icons/hicolor/256x256/apps/${APP_NAME}.png"
  sed "s/@VERSION@/$VERSION/g" packaging/debian/control > "$pkgroot/DEBIAN/control"

  dpkg-deb --build "$pkgroot" "$DIST_DIR/${APP_NAME}_${VERSION}_amd64.deb"
}

build_appimage() {
  [ "$GOOS_HOST" = "linux" ] || die "AppImage build must run on Linux."

  build_linux_gui

  local appdir="$DIST_DIR/appimage/${APP_NAME}.AppDir"
  rm -rf "$appdir"
  mkdir -p "$appdir/usr/bin" "$appdir/usr/share/applications" "$appdir/usr/share/icons/hicolor/256x256/apps"

  install -Dm755 "$DIST_DIR/linux/$APP_NAME" "$appdir/usr/bin/$APP_NAME"
  install -Dm755 packaging/appimage/AppRun "$appdir/AppRun"
  install -Dm644 packaging/linux/${APP_NAME}.desktop "$appdir/${APP_NAME}.desktop"
  install -Dm644 packaging/linux/${APP_NAME}.desktop "$appdir/usr/share/applications/${APP_NAME}.desktop"
  install -Dm644 build/appicon.png "$appdir/${APP_NAME}.png"
  install -Dm644 build/appicon.png "$appdir/usr/share/icons/hicolor/256x256/apps/${APP_NAME}.png"

  if [ -x ./linuxdeploy-x86_64.AppImage ]; then
    ./linuxdeploy-x86_64.AppImage --appdir "$appdir" --output appimage
  elif need_cmd linuxdeploy; then
    linuxdeploy --appdir "$appdir" --output appimage
  else
    warn "linuxdeploy not found. AppDir prepared at $appdir"
    warn "Download linuxdeploy-x86_64.AppImage and rerun: ./build.sh appimage"
  fi
}

build_arch_package() {
  [ "$GOOS_HOST" = "linux" ] || die "Arch package build must run on Linux."

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
url = https://github.com/smyhlin/SimklExpoGter
builddate = $(date +%s)
packager = SimklExpoGter build.sh
size = $(du -sb "$pkgroot" | awk '{print $1}')
arch = x86_64
license = MIT
depend = gtk3
depend = webkit2gtk-4.1
depend = ca-certificates
PKGINFO

  if need_cmd bsdtar; then
    (cd "$pkgroot" && bsdtar --zstd -cf "$package" .)
    log "Arch package written to $package"
  else
    cp packaging/arch/PKGBUILD "$DIST_DIR/arch/PKGBUILD"
    warn "bsdtar not found. Package tree prepared at $pkgroot"
    warn "Reference PKGBUILD copied to $DIST_DIR/arch/PKGBUILD"
  fi
}

archive_tar_gz() {
  local source_dir="$1"
  local source_file="$2"
  local out_file="$3"

  [ -f "$source_dir/$source_file" ] || die "Missing release input: $source_dir/$source_file"
  tar -C "$source_dir" -czf "$out_file" "$source_file"
}

archive_zip() {
  local out_file="$1"
  shift

  if need_cmd zip; then
    zip -j -q "$out_file" "$@"
  else
    python - "$out_file" "$@" <<'PY'
import sys, zipfile, pathlib
out = pathlib.Path(sys.argv[1])
with zipfile.ZipFile(out, "w", zipfile.ZIP_DEFLATED) as z:
    for item in sys.argv[2:]:
        p = pathlib.Path(item)
        z.write(p, p.name)
PY
  fi
}

build_release_linux() {
  [ "$GOOS_HOST" = "linux" ] || die "Linux release must be built on Linux."

  local tag suffix release_dir
  tag="$(detect_webkit_tag)"
  suffix="$(webkit_suffix "$tag")"
  release_dir="$DIST_DIR/release"

  log "Building Linux release assets for version $VERSION"
  mkdir -p "$release_dir"

  build_linux_gui
  build_linux_cli
  build_windows_gui
  build_windows_cli

  archive_tar_gz "$DIST_DIR/linux" "$APP_NAME" "$release_dir/${APP_NAME}-linux-gui-${suffix}-amd64-v${VERSION}.tar.gz"
  archive_tar_gz "$DIST_DIR/linux" "${APP_NAME}-cli" "$release_dir/${APP_NAME}-linux-cli-amd64-v${VERSION}.tar.gz"
  archive_zip "$release_dir/${APP_NAME}-windows-gui-amd64-v${VERSION}.zip" "$DIST_DIR/windows/${APP_NAME}.exe"
  archive_zip "$release_dir/${APP_NAME}-windows-cli-amd64-v${VERSION}.zip" "$DIST_DIR/windows/${APP_NAME}-cli.exe"

  if need_cmd dpkg-deb; then
    build_deb || warn "Debian package failed; release tarballs are still available."
    cp -f "$DIST_DIR/${APP_NAME}_${VERSION}_amd64.deb" "$release_dir/" 2>/dev/null || true
  else
    warn "dpkg-deb not found; skipping .deb package."
  fi

  build_arch_package || warn "Arch package failed; release tarballs are still available."
  cp -f "$DIST_DIR/${APP_NAME}-${VERSION}-1-x86_64.pkg.tar.zst" "$release_dir/" 2>/dev/null || true

  (cd "$release_dir" && sha256sum * > SHA256SUMS.txt)

  log "Release assets:"
  ls -lh "$release_dir"
}

clean() {
  rm -rf "$DIST_DIR" build/bin frontend/dist .build
}

usage() {
  cat <<HELP
Usage: ./build.sh <command>

Commands:
  bootstrap       Run Linux bootstrap
  check           Run shell, frontend and Go checks
  frontend        Build frontend only
  linux           Build Linux GUI binary with Wails
  linux-cli       Build Linux CLI/TUI-only binary, no GUI/WebKit needed
  windows         Cross-build Windows GUI binary with Wails
  windows-cli     Cross-build Windows CLI/TUI-only binary
  deb             Build Debian package
  appimage        Build AppDir/AppImage package
  arch            Build Arch package or prepare package tree
  release-linux   Build Linux + Windows release assets
  release         Alias for release-linux on Linux
  clean           Remove generated build outputs

Environment:
  VERSION=0.1.0              Release/package version
  WAILS_WEBKIT_TAG=webkit2_41 Force Wails WebKitGTK tag
  WAILS_TAGS=webkit2_41       Compatibility alias for WAILS_WEBKIT_TAG
HELP
}

case "${1:-help}" in
  bootstrap) ./scripts/bootstrap.sh "${@:2}" ;;
  check|test) check ;;
  frontend) frontend_build ;;
  linux|linux-gui) build_linux_gui ;;
  linux-cli) build_linux_cli ;;
  windows|windows-gui) build_windows_gui ;;
  windows-cli) build_windows_cli ;;
  deb) build_deb ;;
  appimage) build_appimage ;;
  arch) build_arch_package ;;
  release|release-linux) build_release_linux ;;
  clean) clean ;;
  help|-h|--help) usage ;;
  *) echo "Unknown command: $1" >&2; usage; exit 2 ;;
esac
