#!/usr/bin/env bash
# Exercises every package layout locally and compares it with the canonical
# tree from stage-install.sh. It does not publish, tag or use production keys.
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
source "$root/metadata.sh"
version=$(<"$root/VERSION")

for command in go dpkg-deb diff find sed; do
  command -v "$command" >/dev/null || {
    echo "Missing required command: $command" >&2
    exit 1
  }
done

nfpm=$(command -v nfpm || true)
if [[ -z "$nfpm" && -x "$(go env GOPATH)/bin/nfpm" ]]; then
  nfpm="$(go env GOPATH)/bin/nfpm"
fi
if [[ -z "$nfpm" ]]; then
  echo 'Missing required command: nfpm (install with go install github.com/goreleaser/nfpm/v2/cmd/nfpm@v2.43.0)' >&2
  exit 1
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

cd "$root"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags='-s -w' -o "$tmp/devlinger" .

"$root/scripts/stage-install.sh" system "$tmp/expected-system" "$tmp/devlinger"
"$root/scripts/stage-install.sh" appdir "$tmp/expected-appdir" "$tmp/devlinger"

manifest() {
  (
    cd "$1"
    find . -printf '%P|%y|%m|%l\n' | sort
  )
}

compare_tree() {
  local expected=$1 actual=$2 label=$3
  diff -ruN --no-dereference "$expected" "$actual"
  diff -u <(manifest "$expected") <(manifest "$actual")
  echo "$label layout: OK"
}

# DEB: use the same builder that build-apt.sh publishes.
"$root/scripts/build-deb.sh" "$version" "$tmp/devlinger.deb" "$tmp/devlinger"
mkdir "$tmp/deb-root"
dpkg-deb -x "$tmp/devlinger.deb" "$tmp/deb-root"
compare_tree "$tmp/expected-system" "$tmp/deb-root" DEB

# nFPM: package the canonical tree, then inspect what the package contains.
mkdir -p "$tmp/nfpm-work/dist"
cp "$tmp/devlinger" "$tmp/nfpm-work/dist/devlinger"
"$root/scripts/stage-install.sh" \
  system "$tmp/nfpm-work/dist/package-root" "$tmp/nfpm-work/dist/devlinger"
(
  cd "$tmp/nfpm-work"
  VERSION="$version" REPO_URL="$REPO_URL" \
    "$nfpm" package --config "$root/packaging/nfpm.yaml" \
      --packager deb --target "$tmp/nfpm.deb"
)
mkdir "$tmp/nfpm-root"
dpkg-deb -x "$tmp/nfpm.deb" "$tmp/nfpm-root"
compare_tree "$tmp/expected-system" "$tmp/nfpm-root" nFPM

# AUR: generate and execute the actual PKGBUILD package() function.
mkdir -p "$tmp/aur/src" "$tmp/aur/pkg"
mkdir "$tmp/aur/src/dev.linger-$version"
cp -a "$root/." "$tmp/aur/src/dev.linger-$version/"
rm -rf \
  "$tmp/aur/src/dev.linger-$version/.git" \
  "$tmp/aur/src/dev.linger-$version/devlinger" \
  "$tmp/aur/src/dev.linger-$version/dist"
sed \
  -e "s/@VERSION@/$version/g" \
  -e 's/@SHA256@/not-used-by-layout-test/g' \
  -e "s|@REPO_URL@|$REPO_URL|g" \
  "$root/packaging/aur/PKGBUILD.in" > "$tmp/aur/PKGBUILD"
(
  srcdir="$tmp/aur/src"
  pkgdir="$tmp/aur/pkg"
  source "$tmp/aur/PKGBUILD"
  cp "$tmp/devlinger" "$srcdir/dev.linger-$version/devlinger"
  package
)
compare_tree "$tmp/expected-system" "$tmp/aur/pkg" AUR

# AppImage: exercise the public AppDir builder, including its root symlinks.
bash "$root/scripts/build-appdir.sh" "$tmp/devlinger" "$tmp/appdir" >/dev/null
compare_tree "$tmp/expected-appdir" "$tmp/appdir" AppDir

echo "All package layouts match their canonical install trees"
