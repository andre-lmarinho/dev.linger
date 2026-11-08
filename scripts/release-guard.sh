#!/usr/bin/env bash
# Decide whether this push publishes a release.
# State comes from arguments instead of the repository, so PR CI can test
# every scenario (new release, retry, skip, rollback attempt, bad version).
# Usage: release-guard.sh <version> <head-sha> <tags-file>
#   tags-file: lines of "<tag> <commit-hash>" for existing v* tags
# Prints GITHUB_OUTPUT "key=value" lines on stdout; details go to stderr.

set -euo pipefail

version=${1:?}
head=${2:?}
tags_file=${3:?}

if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Invalid VERSION: $version" >&2
  exit 1
fi

tag="v$version"

echo "version=$version"
echo "tag=$tag"

mapfile -t tags < <(grep -E '^v[0-9]+\.[0-9]+\.[0-9]+ ' "$tags_file" || true)

for entry in "${tags[@]}"; do
  if [[ "${entry%% *}" == "$tag" ]]; then
    tagged_sha=${entry#* }

    if [[ "$tagged_sha" == "$head" ]]; then
      echo "Retrying release $tag" >&2
      echo "should_release=true"
    else
      echo "$tag is already published." >&2
      echo "should_release=false"
    fi

    exit 0
  fi
done

if ((${#tags[@]})); then
  latest=$(
    printf '%s\n' "${tags[@]}" | cut -d ' ' -f 1 | sort -V | tail -1
  )

  newest=$(printf '%s\n' "$latest" "$tag" | sort -V | tail -1)

  if [[ "$newest" != "$tag" ]]; then
    echo "VERSION is older than the latest tag: $latest" >&2
    exit 1
  fi
fi

echo "Publishing new version $tag" >&2
echo "should_release=true"
