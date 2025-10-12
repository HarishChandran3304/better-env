#!/usr/bin/env sh
set -euo pipefail

# Simple installer for bnv (better-env) for macOS/Linux

REPO="HarishChandran3304/better-env"
BIN="bnv"
PREFIX="${BNV_PREFIX:-/usr/local}"

command_exists() { command -v "$1" >/dev/null 2>&1; }

# Check for required tools
if ! command_exists curl && ! command_exists wget; then
  echo "Error: curl or wget is required" >&2
  exit 1
fi

fetch() {
  if command_exists curl; then
    curl -fsSL "$1"
  else
    wget -qO- "$1"
  fi
}

# Detect OS
uname_s=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$uname_s" in
  linux*)   OS=linux ;;
  darwin*)  OS=darwin ;;
  *) echo "Error: unsupported OS: $uname_s" >&2; exit 1 ;;
esac

# Detect Architecture
uname_m=$(uname -m)
case "$uname_m" in
  x86_64|amd64) ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *) echo "Error: unsupported ARCH: $uname_m" >&2; exit 1 ;;
esac

# Fetch latest release
API="https://api.github.com/repos/$REPO/releases/latest"
echo "Fetching latest release metadata…"

if command_exists jq; then
  tag=$(fetch "$API" | jq -r '.tag_name')
else
  tag=$(fetch "$API" | grep '"tag_name"' | head -n1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
fi

if [ -z "$tag" ]; then
  echo "Error: failed to determine latest version" >&2
  exit 1
fi

# Strip 'v' prefix if present (adjust based on your actual release naming)
version=${tag#v}
archive="$BIN"_"$version"_"$OS"_"$ARCH".tar.gz
base_url="https://github.com/$REPO/releases/download/$tag"

# Setup temp directory
tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT INT TERM

# Download files
echo "Downloading ${archive}…"
fetch "${base_url}/${archive}" > "${tmpdir}/${archive}"
fetch "${base_url}/checksums.txt" > "${tmpdir}/checksums.txt"

# Verify checksum
echo "Verifying checksum…"
cd "$tmpdir"
checksum_line=$(grep "[[:space:]]${archive}$" checksums.txt || true)

if [ -z "$checksum_line" ]; then
  echo "Error: checksum for $archive not found in checksums.txt" >&2
  exit 1
fi

expected=$(echo "$checksum_line" | awk '{print $1}')

if command_exists shasum; then
  actual=$(shasum -a 256 "$archive" | awk '{print $1}')
elif command_exists sha256sum; then
  actual=$(sha256sum "$archive" | awk '{print $1}')
else
  echo "Warning: no sha256 utility found; skipping checksum verification" >&2
  actual="$expected"
fi

if [ "$expected" != "$actual" ]; then
  echo "Error: checksum mismatch" >&2
  echo "  Expected: $expected" >&2
  echo "  Got:      $actual" >&2
  exit 1
fi

# Extract
echo "Extracting…"
tar -xzf "${archive}"

if [ ! -f "$BIN" ]; then
  echo "Error: binary '$BIN' not found after extraction" >&2
  exit 1
fi

# Install
install_dir="$PREFIX/bin"
echo "Installing to $install_dir (may require sudo)…"
mkdir -p "$install_dir" 2>/dev/null || true

if [ -w "$install_dir" ]; then
  mv "$BIN" "$install_dir/$BIN"
  chmod +x "$install_dir/$BIN"
else
  sudo mv "$BIN" "$install_dir/$BIN"
  sudo chmod +x "$install_dir/$BIN"
fi

echo "✓ Successfully installed: $install_dir/$BIN"

# Check PATH
if ! echo "$PATH" | grep -q "$install_dir"; then
  echo ""
  echo "⚠️  Warning: $install_dir is not in your PATH"
  echo "   Add it by running:"
  echo "     export PATH=\"$install_dir:\$PATH\""
  echo "   Or add it permanently to your ~/.bashrc or ~/.zshrc"
  echo ""
fi

# Verify installation
if command -v "$BIN" >/dev/null 2>&1; then
  version_output=$("$BIN" --version 2>&1 || echo "installed")
  echo "Version: $version_output"
else
  echo "Note: Run 'source ~/.bashrc' or restart your shell if $BIN is not found"
fi