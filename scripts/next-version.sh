#!/usr/bin/env bash
set -euo pipefail

# Only published tags reachable from this commit are used as the baseline.
tag=$(git tag --merged HEAD --list 'v*' | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -1 || true)
if [[ -z "$tag" ]]; then
  echo 'No stable version tag found' >&2
  exit 1
fi
version=${tag#v}
IFS=. read -r major minor patch <<< "$version"
messages=$(git log --format=%B "$tag"..HEAD)
if grep -Eq '(^|[[:space:]])BREAKING CHANGE:|^[[:alnum:]_-]+(\([^)]*\))?!:' <<< "$messages"; then
  echo "$((major + 1)).0.0"
elif grep -Eq '^feat(\([^)]*\))?:' <<< "$messages"; then
  echo "$major.$((minor + 1)).0"
else
  echo "$major.$minor.$((patch + 1))"
fi
