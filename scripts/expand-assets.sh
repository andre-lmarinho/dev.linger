#!/usr/bin/env bash
# Expands the desktop integration assets from metadata.sh into $1 using the
# canonical AppID, so every package format ships identical files. Pass
# --autostart to also install the autostart entry (DEB and AUR packages;
# AppImages must not ship an autostart pointing at /usr/bin).
set -euo pipefail

dest=
autostart=false
for argument in "$@"; do
  case "$argument" in
    --autostart) autostart=true ;;
    *) dest=$argument ;;
  esac
done

if [[ -z "$dest" ]]; then
  echo "Usage: $0 [--autostart] DESTDIR" >&2
  exit 1
fi

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
source "$root/metadata.sh"

expand() {
  sed -e "s|@APPID@|$APPID|g" -e "s|@REPO_URL@|$REPO_URL|g" \
    "$root/packaging/linux/$1"
}

mkdir -p \
  "$dest/usr/share/applications" \
  "$dest/usr/share/metainfo" \
  "$dest/usr/share/icons/hicolor/scalable/apps"

expand devlinger.desktop.in > "$dest/usr/share/applications/$APPID.desktop"
expand devlinger.metainfo.xml.in > "$dest/usr/share/metainfo/$APPID.metainfo.xml"
install -m 644 "$root/packaging/linux/$APPID.svg" \
  "$dest/usr/share/icons/hicolor/scalable/apps/$APPID.svg"

if $autostart; then
  mkdir -p "$dest/etc/xdg/autostart"
  install -m 644 "$dest/usr/share/applications/$APPID.desktop" \
    "$dest/etc/xdg/autostart/$APPID.desktop"
fi
