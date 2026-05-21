#!/usr/bin/env bash
set -euo pipefail
echo "=== pkgr version ==="
pkgr version

echo "=== pkgr pm list ==="
pkgr pm list

echo "=== pkgr search node (npm + apt) ==="
pkgr search node --limit 5

echo "=== pkgr list ==="
pkgr list --limit 10 || true

echo "=== pkgr --dry-run install ripgrep@apt ==="
pkgr --dry-run install ripgrep@apt

echo "ubuntu smoke OK"
