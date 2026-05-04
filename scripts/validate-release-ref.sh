#!/usr/bin/env bash
set -euo pipefail

tag="${1:?release tag required}"
base_ref="${2:-origin/main}"

if [[ ! "${tag}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Invalid release tag: ${tag}" >&2
  exit 1
fi

git fetch --force origin refs/heads/main:refs/remotes/origin/main --tags
git verify-tag "${tag}" >/dev/null

tag_commit="$(git rev-list -n 1 "${tag}")"
if ! git merge-base --is-ancestor "${tag_commit}" "${base_ref}"; then
  echo "Release tag ${tag} is not reachable from ${base_ref}" >&2
  exit 1
fi
