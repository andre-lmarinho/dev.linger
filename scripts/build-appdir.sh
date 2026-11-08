
#!/usr/bin/env bash
set -euo pipefail

binary="${1:-dist/devlinger}"
appdir="${2:-dist/DevLinger.AppDir}"

if [[ ! -f "$binary" ]]; then
  echo "Binary not found: $binary" >&2
  exit 1
fi

appid="io.github.devlinger.DevLinger"
assets="packaging/linux"

rm -rf "$appdir"

mkdir -p \
  "$appdir/usr/bin" \
  "$appdir/usr/share/applications" \
  "$appdir/usr/share/metainfo" \
  "$appdir/usr/share/icons/hicolor/scalable/apps"

install -m 755 "$binary" "$appdir/usr/bin/devlinger"

# AppImage must not depend on an absolute /usr/bin path.
sed 's|^Exec=/usr/bin/devlinger$|Exec=devlinger|' \
  "$assets/$appid.desktop" \
  > "$appdir/usr/share/applications/$appid.desktop"

install -m 644 "$assets/$appid.metainfo.xml" \
  "$appdir/usr/share/metainfo/$appid.metainfo.xml"

install -m 644 "$assets/$appid.svg" \
  "$appdir/usr/share/icons/hicolor/scalable/apps/$appid.svg"

ln -s "usr/bin/devlinger" "$appdir/AppRun"

ln -s "usr/share/applications/$appid.desktop" \
  "$appdir/$appid.desktop"

ln -s "usr/share/icons/hicolor/scalable/apps/$appid.svg" \
  "$appdir/$appid.svg"

echo "AppDir created: $appdir"
