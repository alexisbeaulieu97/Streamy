#!/usr/bin/env bash
set -euo pipefail

REPO="${REPO:-github.com/alexisbeaulieu97/streamy}"
VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
TEMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TEMP_DIR"' EXIT

OS="$(uname | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

if [[ "$OS" == "darwin" ]]; then
  OS="darwin"
elif [[ "$OS" == "linux" ]]; then
  OS="linux"
elif [[ "$OS" == "mingw"* || "$OS" == "msys"* || "$OS" == "cygwin"* ]]; then
  OS="windows"
else
  echo "Unsupported OS: $OS"
  exit 1
fi

if [[ "$VERSION" == "latest" ]]; then
  VERSION=$(curl -sSfL "https://api.github.com/repos/${REPO}/releases/latest" | grep -o '"tag_name":"v[^"]*' | head -n1 | cut -d'"' -f4)
fi

if [[ -z "$VERSION" ]]; then
  echo "Unable to determine release version"
  exit 1
fi

ASSET="streamy-${OS}-${ARCH}"
if [[ "$OS" == "windows" ]]; then
  ASSET+=".exe"
fi

TARBALL_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET}"
TARGET="$TEMP_DIR/$ASSET"

echo "Downloading $TARBALL_URL"
curl -sSfL "$TARBALL_URL" -o "$TARGET"
chmod +x "$TARGET"

mkdir -p "$INSTALL_DIR"
if [[ "$OS" == "windows" ]]; then
  INSTALL_PATH="${INSTALL_DIR}/streamy.exe"
else
  INSTALL_PATH="${INSTALL_DIR}/streamy"
fi

mv "$TARGET" "$INSTALL_PATH"
echo "Installed streamy to $INSTALL_PATH"
