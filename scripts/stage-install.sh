#!/usr/bin/env bash
# Builds the complete install tree consumed by every package format.
set -euo pipefail

mode="${1:?Usage: $0 system|appdir DESTDIR BINARY}"
dest="${2:?Usage: $0 system|appdir DESTDIR BINARY}"
binary="${3:?Usage: $0 system|appdir DESTDIR BINARY}"

if [[ "$mode" != system && "$mode" != appdir ]]; then
  echo "Unknown install layout: $mode" >&2
  exit 1
fi
if [[ ! -f "$binary" ]]; then
  echo "Binary not found: $binary" >&2
  exit 1
fi

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
source "$root/metadata.sh"

rm -rf "$dest"
mkdir -p "$dest/usr/bin" "$dest/usr/share/licenses/devlinger"
install -m 755 "$binary" "$dest/usr/bin/devlinger"
install -m 644 "$root/LICENSE" "$dest/usr/share/licenses/devlinger/LICENSE"

if [[ "$mode" == system ]]; then
  "$root/scripts/expand-assets.sh" --autostart "$dest"
  exit
fi

"$root/scripts/expand-assets.sh" "$dest"

# AppImage launches from its own root and must not depend on /usr/bin.
sed -i 's|^Exec=/usr/bin/devlinger$|Exec=devlinger|' \
  "$dest/usr/share/applications/$APPID.desktop"

ln -s "usr/bin/devlinger" "$dest/AppRun"
ln -s "usr/share/applications/$APPID.desktop" "$dest/$APPID.desktop"
ln -s "usr/share/icons/hicolor/scalable/apps/$APPID.svg" "$dest/$APPID.svg"
