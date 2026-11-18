#!/usr/bin/env bash
set -euo pipefail

binary="${1:-dist/devlinger}"
appdir="${2:-dist/DevLinger.AppDir}"

if [[ ! -f "$binary" ]]; then
  echo "Binary not found: $binary" >&2
  exit 1
fi

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
source "$root/metadata.sh"

rm -rf "$appdir"
mkdir -p "$appdir/usr/bin"

install -m 755 "$binary" "$appdir/usr/bin/devlinger"

"$root/scripts/expand-assets.sh" "$appdir"

# AppImage must not depend on an absolute /usr/bin path.
sed -i 's|^Exec=/usr/bin/devlinger$|Exec=devlinger|' \
  "$appdir/usr/share/applications/$APPID.desktop"

ln -s "usr/bin/devlinger" "$appdir/AppRun"

ln -s "usr/share/applications/$APPID.desktop" \
  "$appdir/$APPID.desktop"

ln -s "usr/share/icons/hicolor/scalable/apps/$APPID.svg" \
  "$appdir/$APPID.svg"

echo "AppDir created: $appdir"
