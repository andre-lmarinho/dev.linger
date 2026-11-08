#!/usr/bin/env bash
set -euo pipefail

url=https://andre-lmarinho.github.io/dev.linger
fingerprint=A17FD32EBCB92EC0C06CBA8E1B2A90F1DB7748F8
for command in curl gpg sudo dpkg apt-get; do
  command -v "$command" >/dev/null || { echo "Missing required command: $command" >&2; exit 1; }
done
if [[ $(dpkg --print-architecture) != amd64 ]]; then
  echo 'DevLinger currently supports Debian/Ubuntu amd64 only.' >&2
  exit 1
fi
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
curl -fsSL "$url/devlinger.gpg" -o "$tmp/devlinger.gpg"
keys=$(gpg --batch --show-keys --with-colons "$tmp/devlinger.gpg" | awk -F: '$1 == "pub" { primary=1; next } primary && $1 == "fpr" { print $10; primary=0 }')
if [[ "$keys" != "$fingerprint" ]]; then
  echo 'The repository signing key fingerprint does not match.' >&2
  exit 1
fi
printf '%s\n' "deb [arch=amd64 signed-by=/usr/share/keyrings/devlinger.gpg] $url stable main" > "$tmp/devlinger.list"
sudo install -m 644 "$tmp/devlinger.gpg" /usr/share/keyrings/devlinger.gpg
sudo install -m 644 "$tmp/devlinger.list" /etc/apt/sources.list.d/devlinger.list
sudo apt-get update
sudo apt-get install -y devlinger
