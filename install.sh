#!/usr/bin/env bash

set -e

REPO="plumthedev/myrient-cli"
BIN_NAME="myrient-cli"
INSTALL_DIR="${HOME}/.local/bin"

# Try sudo install if available
if command -v sudo >/dev/null && [ -w /usr/local/bin ]; then
  INSTALL_DIR="/usr/local/bin"
fi

# Detect OS
OS="$(uname | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  armv7l|armv6l) ARCH="arm" ;;
  *) echo "❌ Unsupported architecture: $ARCH"; exit 1 ;;
esac

EXT=""
if [[ "$OS" == "windows" || "$OS" == "mingw"* || "$OS" == "msys"* ]]; then
  EXT=".exe"
fi

# Get latest version tag from GitHub API
LATEST=$(curl -s https://api.github.com/repos/$REPO/releases/latest | grep tag_name | cut -d '"' -f 4)
if [ -z "$LATEST" ]; then
  echo "❌ Failed to fetch latest version info."
  exit 1
fi

FILENAME="${BIN_NAME}-${OS}-${ARCH}${EXT}"
URL="https://github.com/${REPO}/releases/download/${LATEST}/${FILENAME}"

echo "⬇️  Downloading ${FILENAME} from ${URL}"
curl -L "$URL" -o "/tmp/${FILENAME}"

echo "📦 Installing to ${INSTALL_DIR}..."
mkdir -p "$INSTALL_DIR"
chmod +x "/tmp/${FILENAME}"
mv "/tmp/${FILENAME}" "${INSTALL_DIR}/myrient"

echo "✅ Installed 'myrient' to ${INSTALL_DIR}"

if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
  echo "⚠️  $INSTALL_DIR is not in your PATH."
  echo "👉 Add this line to your shell config:"
  echo "export PATH=\"\$PATH:$INSTALL_DIR\""
fi

echo "🎉 Done! Run with: myrient"
