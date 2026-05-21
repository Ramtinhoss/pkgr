#!/usr/bin/env bash
set -euo pipefail
pkgr version
pkgr pm list
pkgr search ripgrep --pm pacman --limit 5
pkgr --dry-run install ripgrep@pacman
echo "arch smoke OK"
