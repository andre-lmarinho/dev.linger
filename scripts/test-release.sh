#!/usr/bin/env bash
set -euo pipefail
root=$(git rev-parse --show-toplevel)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

git init -q "$tmp/versions"
git -C "$tmp/versions" config user.name test
git -C "$tmp/versions" config user.email test@example.org
git -C "$tmp/versions" commit -q --allow-empty -m initial
git -C "$tmp/versions" tag v0.1.0
git -C "$tmp/versions" commit -q --allow-empty -m 'docs: update'
[[ $(cd "$tmp/versions" && "$root/scripts/next-version.sh") == 0.1.1 ]]
git -C "$tmp/versions" commit -q --allow-empty -m 'fix: patch'
[[ $(cd "$tmp/versions" && "$root/scripts/next-version.sh") == 0.1.1 ]]
git -C "$tmp/versions" commit -q --allow-empty -m 'feat(tray): feature'
[[ $(cd "$tmp/versions" && "$root/scripts/next-version.sh") == 0.2.0 ]]
git -C "$tmp/versions" commit -q --allow-empty -m 'feat!: breaking'
[[ $(cd "$tmp/versions" && "$root/scripts/next-version.sh") == 1.0.0 ]]

export GNUPGHOME="$tmp/gnupg"
mkdir -m 700 "$GNUPGHOME"
gpg --batch --passphrase '' --quick-generate-key 'DevLinger test <test@example.org>' ed25519 sign 1d >/dev/null 2>&1
key=$(gpg --batch --with-colons --list-secret-keys | awk -F: '$1 == "fpr" { print $10; exit }')
touch "$tmp/passphrase"
"$root/scripts/build-apt.sh" 0.1.0 "$tmp/repo" "$key" "$tmp/passphrase"
"$root/scripts/build-apt.sh" 0.1.1 "$tmp/repo" "$key" "$tmp/passphrase"
gpg --batch --verify "$tmp/repo/dists/stable/InRelease" >/dev/null 2>&1
mkdir -p "$tmp/apt/lists/partial" "$tmp/apt/cache/archives/partial" "$tmp/apt/etc"
printf 'deb [signed-by=%s/repo/devlinger.gpg] file:%s/repo stable main\n' "$tmp" "$tmp" > "$tmp/apt/etc/sources.list"
cat > "$tmp/apt/status" <<'EOF'
Package: devlinger
Status: install ok installed
Priority: optional
Section: utils
Installed-Size: 1
Maintainer: DevLinger contributors
Architecture: amd64
Version: 0.1.0
Description: Linux system tray monitor for local development services
EOF
opts=(-o "Dir::Etc::sourcelist=$tmp/apt/etc/sources.list" -o Dir::Etc::sourceparts=- \
  -o "Dir::State::lists=$tmp/apt/lists" -o "Dir::State::status=$tmp/apt/status" \
  -o "Dir::Cache=$tmp/apt/cache" -o APT::Sandbox::User=root -o APT::Architecture=amd64)
apt-get "${opts[@]}" update >/dev/null
apt-get "${opts[@]}" -s upgrade | grep -q 'Inst devlinger \[0.1.0\] (0.1.1 '
(cd "$tmp" && apt-get "${opts[@]}" download devlinger >/dev/null)
[[ $(dpkg-deb -f "$tmp/devlinger_0.1.1_amd64.deb" Version) == 0.1.1 ]]
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
chmod +x "$tmp/mock/"*
export TEST_SUDO_LOG="$tmp/sudo.log"
PATH="$tmp/mock:$PATH" bash "$root/scripts/install.sh"
grep -q '^apt-get install -y devlinger$' "$TEST_SUDO_LOG"
: > "$TEST_SUDO_LOG"
if TEST_FINGERPRINT=BAD PATH="$tmp/mock:$PATH" bash "$root/scripts/install.sh" 2>/dev/null; then
  echo 'Installer accepted an unexpected signing key' >&2
  exit 1
fi
[[ ! -s "$TEST_SUDO_LOG" ]]
echo 'Versioning and signed APT upgrade tests passed'
