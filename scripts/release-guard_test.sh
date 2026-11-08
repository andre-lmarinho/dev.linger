#!/usr/bin/env bash
set -euo pipefail

scripts=$(cd "$(dirname "$0")" && pwd)
guard="$scripts/release-guard.sh"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

tags="$tmp/tags"

# Expect the guard to succeed and print exactly these GITHUB_OUTPUT lines.
expect_ok() {
  # name, version, head sha, tags content, expected stdout
  local name=${1:?} version=${2:?} head=${3:?} state=${4-} expected=${5:?}

  printf '%s\n' "$state" > "$tags"

  local out
  if ! out=$("$guard" "$version" "$head" "$tags" 2>/dev/null); then
    echo "FAIL $name: guard exited with error" >&2
    exit 1
  fi

  if [[ "$out" != "$expected" ]]; then
    echo "FAIL $name: expected '$expected'" >&2
    echo "FAIL $name: got '$out'" >&2
    exit 1
  fi

  echo "ok: $name"
}

expect_fail() {
  # name, version, head sha, tags content
  local name=${1:?} version=${2:?} head=${3:?} state=${4-}

  printf '%s\n' "$state" > "$tags"

  if "$guard" "$version" "$head" "$tags" >/dev/null 2>&1; then
    echo "FAIL $name: guard accepted an invalid state" >&2
    exit 1
  fi

  echo "ok: $name"
}

expect_ok 'publishes the first release' \
  0.1.0 aaaabbbb '' \
  $'version=0.1.0\ntag=v0.1.0\nshould_release=true'

expect_ok 'publishes a newer VERSION than the latest tag' \
  0.3.2 bbbbcccc "$(printf 'v0.3.1 ccccdddd')" \
  $'version=0.3.2\ntag=v0.3.2\nshould_release=true'

expect_ok 'retries when the tag points at HEAD' \
  0.3.2 ddddeeee "$(printf 'v0.3.1 aaaabbbb\nv0.3.2 ddddeeee')" \
  $'version=0.3.2\ntag=v0.3.2\nshould_release=true'

expect_ok 'skips when the tag points at another commit' \
  0.3.2 eeeeffff "$(printf 'v0.3.2 ffffgggg')" \
  $'version=0.3.2\ntag=v0.3.2\nshould_release=false'

expect_ok 'ignores non-semver v tags for monotonicity' \
  0.2.2 aaaabbbb "$(printf 'vbanana ccccdddd\nv0.2.1 ddddeeee')" \
  $'version=0.2.2\ntag=v0.2.2\nshould_release=true'

expect_fail 'rejects a VERSION older than the latest tag: rollback' \
  0.3.2 bbbbaaaa "$(printf 'v0.3.3 ccccdddd')"

expect_fail 'rejects a non-semver VERSION' \
  0.3 aaaabbbb "$(printf 'v0.2.0 ccccdddd')"

echo 'release guard scenario tests passed'
