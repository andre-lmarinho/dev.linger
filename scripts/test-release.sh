
#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# Validate the repository's explicit release version.
version=$(<"$root/VERSION")

if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Invalid VERSION: $version" >&2
  exit 1
fi

# Use a fixed historical version to test an actual APT upgrade.
installed_version="0.1.0"

if [[ "$version" == "$installed_version" ]]; then
  echo "VERSION must be newer than $installed_version for upgrade testing" >&2
  exit 1
fi

export GNUPGHOME="$tmp/gnupg"
mkdir -m 700 "$GNUPGHOME"

gpg --batch --passphrase '' \
  --quick-generate-key 'DevLinger test <test@example.org>' \
  ed25519 sign 1d >/dev/null 2>&1

key=$(
  gpg --batch --with-colons --list-secret-keys |
    awk -F: '$1 == "fpr" { print $10; exit }'
)

touch "$tmp/passphrase"

# Build both versions to verify repository upgrades.
"$root/scripts/build-apt.sh" \
  "$installed_version" "$tmp/repo" "$key" "$tmp/passphrase"

"$root/scripts/build-apt.sh" \
  "$version" "$tmp/repo" "$key" "$tmp/passphrase"

gpg --batch --verify \
  "$tmp/repo/dists/stable/InRelease" >/dev/null 2>&1

mkdir -p \
  "$tmp/apt/lists/partial" \
  "$tmp/apt/cache/archives/partial" \
  "$tmp/apt/etc"

printf 'deb [signed-by=%s/repo/devlinger.gpg] file:%s/repo stable main\n' \
  "$tmp" "$tmp" > "$tmp/apt/etc/sources.list"

cat > "$tmp/apt/status" <<EOF
Package: devlinger
Status: install ok installed
Priority: optional
Section: utils
Installed-Size: 1
Maintainer: DevLinger contributors
Architecture: amd64
Version: $installed_version
Description: Linux system tray monitor for local development services
EOF

opts=(
  -o "Dir::Etc::sourcelist=$tmp/apt/etc/sources.list"
  -o Dir::Etc::sourceparts=-
  -o "Dir::State::lists=$tmp/apt/lists"
  -o "Dir::State::status=$tmp/apt/status"
  -o "Dir::Cache=$tmp/apt/cache"
  -o APT::Sandbox::User=root
  -o APT::Architecture=amd64
)

apt-get "${opts[@]}" update >/dev/null

apt-get "${opts[@]}" -s upgrade |
  grep -Fq "Inst devlinger [$installed_version] ($version "

(
  cd "$tmp"
  apt-get "${opts[@]}" download devlinger >/dev/null
)

package="$tmp/devlinger_${version}_amd64.deb"

[[ $(dpkg-deb -f "$package" Version) == "$version" ]]

package_root="$tmp/package-root"
mkdir -p "$package_root"

dpkg-deb -x "$package" "$package_root"

desktop="io.github.devlinger.DevLinger.desktop"
assets="$root/packaging/linux"

for destination in \
  "etc/xdg/autostart/$desktop" \
  "usr/share/applications/$desktop"
do
  cmp "$assets/$desktop" "$package_root/$destination"
done

cmp "$assets/io.github.devlinger.DevLinger.metainfo.xml" \
  "$package_root/usr/share/metainfo/io.github.devlinger.DevLinger.metainfo.xml"

cmp "$assets/io.github.devlinger.DevLinger.svg" \
  "$package_root/usr/share/icons/hicolor/scalable/apps/io.github.devlinger.DevLinger.svg"

cmp "$root/scripts/install.sh" "$tmp/repo/install.sh"

mkdir "$tmp/mock"

cat > "$tmp/mock/dpkg" <<'EOF'
#!/bin/sh
echo amd64
EOF

cat > "$tmp/mock/curl" <<'EOF'
#!/bin/sh
while [ "$1" != -o ]; do shift; done
touch "$2"
EOF

cat > "$tmp/mock/gpg" <<'EOF'
#!/bin/sh
printf 'pub:::::::::\nfpr:::::::::%s:\n' "${TEST_FINGERPRINT:-A17FD32EBCB92EC0C06CBA8E1B2A90F1DB7748F8}"
EOF

cat > "$tmp/mock/sudo" <<'EOF'
#!/bin/sh
printf '%s\n' "$*" >> "$TEST_SUDO_LOG"
EOF

cat > "$tmp/mock/pgrep" <<'EOF'
#!/bin/sh
exit "${TEST_RUNNING:-1}"
EOF

cat > "$tmp/mock/nohup" <<'EOF'
#!/bin/sh
printf '%s\n' "$*" > "$TEST_LAUNCH_LOG"
EOF

chmod +x "$tmp/mock/"*

export TEST_SUDO_LOG="$tmp/sudo.log"
export TEST_LAUNCH_LOG="$tmp/launch.log"

DISPLAY=:99 WAYLAND_DISPLAY= PATH="$tmp/mock:$PATH" \
  bash "$root/scripts/install.sh"

for i in {1..50}; do
  [[ -s "$TEST_LAUNCH_LOG" ]] && break
  sleep 0.02
done

grep -qx devlinger "$TEST_LAUNCH_LOG"
grep -q '^apt-get install -y devlinger$' "$TEST_SUDO_LOG"

rm "$TEST_LAUNCH_LOG"

TEST_RUNNING=0 DISPLAY=:99 PATH="$tmp/mock:$PATH" \
  bash "$root/scripts/install.sh" |
  grep -q 'already running'

[[ ! -e "$TEST_LAUNCH_LOG" ]]

DISPLAY= WAYLAND_DISPLAY= PATH="$tmp/mock:$PATH" \
  bash "$root/scripts/install.sh" |
  grep -q 'No desktop session'

[[ ! -e "$TEST_LAUNCH_LOG" ]]

: > "$TEST_SUDO_LOG"

if TEST_FINGERPRINT=BAD PATH="$tmp/mock:$PATH" \
  bash "$root/scripts/install.sh" 2>/dev/null; then
  echo 'Installer accepted an unexpected signing key' >&2
  exit 1
fi

[[ ! -s "$TEST_SUDO_LOG" ]]

echo 'Explicit versioning and signed APT upgrade tests passed'
