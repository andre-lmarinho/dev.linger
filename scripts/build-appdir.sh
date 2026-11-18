#!/usr/bin/env bash
set -euo pipefail

binary="${1:-dist/devlinger}"
appdir="${2:-dist/DevLinger.AppDir}"

if [[ ! -f "$binary" ]]; then
  echo "Binary not found: $binary" >&2
  exit 1
fi

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
"$root/scripts/stage-install.sh" appdir "$appdir" "$binary"

echo "AppDir created: $appdir"
