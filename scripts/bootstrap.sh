#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

INSTALL_SYSTEM_DEPS=1
INSTALL_WAILS=1
RUN_FRONTEND=1
RUN_TIDY=1
SYSTEM_UPGRADE=0

for arg in "$@"; do
  case "$arg" in
    --no-system-deps) INSTALL_SYSTEM_DEPS=0 ;;
    --no-wails) INSTALL_WAILS=0 ;;
    --no-frontend) RUN_FRONTEND=0 ;;
    --no-tidy) RUN_TIDY=0 ;;
    --system-upgrade) SYSTEM_UPGRADE=1 ;;
    -h|--help)
      cat <<'HELP'
Usage: ./scripts/bootstrap.sh [options]

Installs local build dependencies and prepares the project.

Options:
  --no-system-deps  Skip apt/pacman package installation
  --no-wails        Skip Wails CLI installation
  --no-frontend     Skip frontend npm install
  --no-tidy         Skip go mod tidy/download
  --system-upgrade  Allow distro package upgrade before dependency install
HELP
      exit 0
      ;;
    *) echo "Unknown option: $arg" >&2; exit 2 ;;
  esac
done

need_cmd() {
  command -v "$1" >/dev/null 2>&1
}

sudo_if_needed() {
  if [ "${EUID:-$(id -u)}" -eq 0 ]; then
    "$@"
  else
    sudo "$@"
  fi
}

install_debian_deps() {
  sudo_if_needed apt-get update
  sudo_if_needed apt-get install -y \
    build-essential \
    ca-certificates \
    curl \
    git \
    nodejs \
    npm \
    pkg-config \
    libgtk-3-dev \
    libglib2.0-dev \
    libayatana-appindicator3-dev

  if apt-cache show libwebkit2gtk-4.1-dev >/dev/null 2>&1; then
    sudo_if_needed apt-get install -y libwebkit2gtk-4.1-dev
  else
    sudo_if_needed apt-get install -y libwebkit2gtk-4.0-dev
  fi
}

install_arch_deps() {
  local webkit_pkg="webkit2gtk-4.1"
  if ! pacman -Si "$webkit_pkg" >/dev/null 2>&1; then
    webkit_pkg="webkit2gtk"
  fi

  if [ "$SYSTEM_UPGRADE" -eq 1 ]; then
    sudo_if_needed pacman -Syu --needed --noconfirm \
      base-devel \
      ca-certificates \
      curl \
      git \
      go \
      gtk3 \
      nodejs \
      npm \
      pkgconf \
      "$webkit_pkg"
    return
  fi

  sudo_if_needed pacman -S --needed --noconfirm \
    base-devel \
    ca-certificates \
    curl \
    git \
    go \
    gtk3 \
    nodejs \
    npm \
    pkgconf \
    "$webkit_pkg"
}

if [ "$INSTALL_SYSTEM_DEPS" -eq 1 ]; then
  if need_cmd apt-get; then
    install_debian_deps
  elif need_cmd pacman; then
    install_arch_deps
  else
    echo "Unsupported package manager. Install Go, Node.js/npm, gcc, pkg-config, GTK3 and WebKitGTK manually." >&2
  fi
fi

if ! need_cmd go; then
  echo "Go is not installed or not on PATH. Install Go 1.25+ and rerun this script." >&2
  exit 3
fi

if ! need_cmd npm; then
  echo "npm is not installed or not on PATH. Install Node.js/npm and rerun this script." >&2
  exit 3
fi

mkdir -p .build frontend/dist
: > frontend/dist/.gitkeep

if pkg-config --exists webkit2gtk-4.1 2>/dev/null; then
  echo 'WAILS_WEBKIT_TAG=webkit2_41' > .build/webkit.env
else
  echo 'WAILS_WEBKIT_TAG=' > .build/webkit.env
fi

if [ "$INSTALL_WAILS" -eq 1 ]; then
  go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0
fi

if [ "$RUN_TIDY" -eq 1 ]; then
  go mod tidy
  go mod download
fi

if [ "$RUN_FRONTEND" -eq 1 ]; then
  (cd frontend && npm ci)
fi

if command -v wails >/dev/null 2>&1; then
  wails doctor || true
else
  echo "Wails CLI was installed into GOPATH/bin. Add it to PATH if 'wails' is still unavailable."
  echo "Typical PATH addition: export PATH=\"$(go env GOPATH)/bin:\$PATH\""
fi

echo "Bootstrap complete. Try: ./build.sh check"
