#!/usr/bin/env bash
set -euo pipefail

if [[ "${GITHUB_REF:-}" != refs/heads/main ]]; then
  echo 'Publishing is restricted to main' >&2
  exit 1
fi
: "${SIGNING_KEY:?Configure APT_SIGNING_KEY}"
: "${SIGNING_PASSPHRASE:?Configure APT_SIGNING_PASSPHRASE}"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
export GNUPGHOME="$tmp/gnupg"
mkdir -m 700 "$GNUPGHOME"
printf '%s' "$SIGNING_KEY" | gpg --batch --import
printf '%s' "$SIGNING_PASSPHRASE" > "$tmp/passphrase"
chmod 600 "$tmp/passphrase"
key=$(gpg --batch --with-colons --list-secret-keys | awk -F: '$1 == "sec" { primary=1; next } primary && $1 == "fpr" { print $10; primary=0 }')
if [[ "$key" != A17FD32EBCB92EC0C06CBA8E1B2A90F1DB7748F8 ]]; then
  echo 'Expected exactly the signing key used by the installer' >&2
  exit 1
fi

version=$(<VERSION)

if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Invalid VERSION: $version" >&2
  exit 1
fi

tag="v$version"

# Allow retries only when the published tag matches VERSION and HEAD.
existing=$(git tag --points-at HEAD | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -1 || true)

if [[ -n "$existing" ]]; then
  if [[ "$existing" != "$tag" ]]; then
    echo "VERSION $version does not match existing tag $existing" >&2
    exit 1
  fi

  if git ls-remote --exit-code --tags origin "refs/tags/$tag" >/dev/null; then
    exit 0
  fi
fi

if git ls-remote --exit-code --tags origin "refs/tags/$tag" >/dev/null; then
  echo "Tag $tag already exists on another commit" >&2
  exit 1
fi

repo="$tmp/repo"
git init -q "$repo"
git -C "$repo" config user.name 'github-actions[bot]'
git -C "$repo" config user.email '41898282+github-actions[bot]@users.noreply.github.com'
if git ls-remote --exit-code --heads origin gh-pages >/dev/null; then
  git fetch -q origin refs/heads/gh-pages
  git -C "$repo" fetch -q "$PWD" "$(git rev-parse FETCH_HEAD)"
  git -C "$repo" checkout -q -b gh-pages FETCH_HEAD
  oldkey=$(gpg --show-keys --with-colons "$repo/devlinger.gpg" | awk -F: '$1 == "pub" { primary=1; next } primary && $1 == "fpr" { print $10; exit }')
  if [[ "$oldkey" != "$key" ]]; then
    echo 'Signing key differs from the published key; key rotation requires a migration' >&2
    exit 1
  fi
else
  git -C "$repo" checkout -q --orphan gh-pages
fi
scripts/build-apt.sh "$version" "$repo" "$key" "$tmp/passphrase"
git -C "$repo" add .
git -C "$repo" commit -qm "Publish $tag"

# Publish the repository and the version tag together or not at all.
git tag "$tag" "$GITHUB_SHA"
git fetch -q "$repo" gh-pages
git push --atomic origin "FETCH_HEAD:refs/heads/gh-pages" "refs/tags/$tag:refs/tags/$tag"
