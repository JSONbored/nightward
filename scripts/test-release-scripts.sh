#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if "${repo_root}/scripts/smoke-release-archive.sh" "latest" >/dev/null 2>&1; then
  echo "expected smoke-release-archive.sh to reject non-semver tag" >&2
  exit 1
fi

if "${repo_root}/scripts/verify-npm-release.sh" "v0.1.0" >/dev/null 2>&1; then
  echo "expected verify-npm-release.sh to reject v-prefixed npm version" >&2
  exit 1
fi

echo "release script fixture tests passed."
