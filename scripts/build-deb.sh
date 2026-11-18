#!/usr/bin/env bash
set -euo pipefail

version="${1:?Usage: $0 VERSION OUTPUT BINARY}"
output="${2:?Usage: $0 VERSION OUTPUT BINARY}"
binary="${3:?Usage: $0 VERSION OUTPUT BINARY}"

if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Invalid version: $version" >&2
  exit 1
fi

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

"$root/scripts/stage-install.sh" system "$tmp/package" "$binary"
mkdir -p "$tmp/package/DEBIAN" "$(dirname "$output")"

cat > "$tmp/package/DEBIAN/control" <<EOF
Package: devlinger
Version: $version
Architecture: amd64
Maintainer: DevLinger contributors
Description: Linux system tray monitor for local development services
EOF

dpkg-deb --build --root-owner-group "$tmp/package" "$output"
