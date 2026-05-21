#!/usr/bin/env bash
# Install latest pkgr binary into /usr/local/bin (or $PKGR_INSTALL_DIR).
set -euo pipefail

REPO="ramtinhoss/pkgr"
INSTALL_DIR="${PKGR_INSTALL_DIR:-/usr/local/bin}"

uname_s=$(uname -s)
uname_m=$(uname -m)

case "$uname_s" in
  Linux)   os=linux ;;
  Darwin)  os=darwin ;;
  *) echo "unsupported OS: $uname_s" >&2; exit 1 ;;
esac
case "$uname_m" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) echo "unsupported arch: $uname_m" >&2; exit 1 ;;
esac

# Resolve latest release tag.
tag=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | \
      sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1)
if [ -z "$tag" ]; then
  echo "could not resolve latest tag" >&2; exit 1
fi
ver="${tag#v}"

archive="pkgr_${ver}_${os}_${arch}.tar.gz"
url="https://github.com/${REPO}/releases/download/${tag}/${archive}"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "downloading $url"
curl -fsSL "$url" -o "$tmp/$archive"

# Verify checksum.
sums_url="https://github.com/${REPO}/releases/download/${tag}/checksums.txt"
curl -fsSL "$sums_url" -o "$tmp/checksums.txt"
( cd "$tmp" && grep "$archive" checksums.txt | sha256sum -c - )

# Extract + install.
tar -xzf "$tmp/$archive" -C "$tmp"

if [ -w "$INSTALL_DIR" ]; then
  mv "$tmp/pkgr" "$INSTALL_DIR/pkgr"
else
  sudo mv "$tmp/pkgr" "$INSTALL_DIR/pkgr"
fi
chmod +x "$INSTALL_DIR/pkgr"

echo "installed: $("$INSTALL_DIR/pkgr" version | head -1)"
