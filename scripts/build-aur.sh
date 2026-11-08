#!/usr/bin/env bash
set -euo pipefail

version="${1:?Usage: scripts/build-aur.sh VERSION}"

if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Invalid version: $version" >&2
  exit 1
fi

url="https://github.com/andre-lmarinho/dev.linger/archive/refs/tags/v${version}.tar.gz"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

curl --fail --location --retry 3 \
  --output "$tmp/source.tar.gz" \
  "$url"

checksum=$(sha256sum "$tmp/source.tar.gz" | cut -d ' ' -f 1)

mkdir -p dist/aur

sed \
  -e "s/@VERSION@/$version/g" \
  -e "s/@SHA256@/$checksum/g" \
  packaging/aur/PKGBUILD.in > dist/aur/PKGBUILD

bash -n dist/aur/PKGBUILD

echo "Generated PKGBUILD for v$version"
