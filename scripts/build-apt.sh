#!/usr/bin/env bash
set -euo pipefail

version=$1
repo=$2
key=$3
passphrase_file=$4
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
mkdir -p "$repo/pool" "$repo/dists/stable/main/binary-amd64"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/package/DEBIAN" "$tmp/package/usr/bin" "$tmp/package/etc/xdg/autostart"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o "$tmp/package/usr/bin/devlinger" .
cat > "$tmp/package/etc/xdg/autostart/devlinger.desktop" <<'EOF'
[Desktop Entry]
Name=DevLinger
Comment=Linux system tray monitor for local development services
Exec=devlinger
Type=Application
Categories=System;Monitor;
Hidden=false
EOF
cat > "$tmp/package/DEBIAN/control" <<EOF
Package: devlinger
Version: $version
Architecture: amd64
Maintainer: DevLinger contributors
Description: Linux system tray monitor for local development services
EOF
dpkg-deb --build --root-owner-group "$tmp/package" "$repo/pool/devlinger_${version}_amd64.deb"
install -m 644 scripts/install.sh "$repo/install.sh"
(
  cd "$repo"
  dpkg-scanpackages --multiversion pool /dev/null > dists/stable/main/binary-amd64/Packages
  gzip -n -9 -c dists/stable/main/binary-amd64/Packages > dists/stable/main/binary-amd64/Packages.gz
  apt-ftparchive -o APT::FTPArchive::Release::Origin=DevLinger \
    -o APT::FTPArchive::Release::Label=DevLinger \
    -o APT::FTPArchive::Release::Suite=stable \
    -o APT::FTPArchive::Release::Codename=stable \
    -o APT::FTPArchive::Release::Architectures=amd64 \
    -o APT::FTPArchive::Release::Components=main \
    release dists/stable > dists/stable/Release
  gpg --batch --yes --pinentry-mode loopback --passphrase-file "$passphrase_file" --local-user "$key" --clearsign -o dists/stable/InRelease dists/stable/Release
  gpg --batch --yes --pinentry-mode loopback --passphrase-file "$passphrase_file" --local-user "$key" --armor --detach-sign -o dists/stable/Release.gpg dists/stable/Release
  gpg --batch --export "$key" > devlinger.gpg
  touch .nojekyll
)
