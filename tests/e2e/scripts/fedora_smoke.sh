#!/usr/bin/env bash
set -euo pipefail
pkgr version
pkgr pm list
pkgr search ripgrep --pm dnf --limit 5
pkgr --dry-run install ripgrep@dnf
echo "fedora smoke OK"
