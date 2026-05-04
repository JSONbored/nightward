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

if "${repo_root}/scripts/validate-release-ref.sh" "latest" >/dev/null 2>&1; then
  echo "expected validate-release-ref.sh to reject non-semver tag" >&2
  exit 1
fi

grep -q "git verify-tag" "${repo_root}/scripts/validate-release-ref.sh"
grep -q "merge-base --is-ancestor" "${repo_root}/scripts/validate-release-ref.sh"
if [[ "$(grep -c "validate-release-ref.sh" "${repo_root}/.github/workflows/release.yml")" -lt 2 ]]; then
  echo "expected release workflow to enforce release ref validation before publishing" >&2
  exit 1
fi

echo "release script fixture tests passed."
