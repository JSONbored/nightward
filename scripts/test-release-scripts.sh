#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "${tmp}"' EXIT

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
grep -q "ubuntu-24.04-arm" "${repo_root}/.github/workflows/release.yml"
grep -q "macos-15-intel" "${repo_root}/.github/workflows/release.yml"
grep -q "aarch64-unknown-linux-gnu" "${repo_root}/.github/workflows/release.yml"
grep -q "x86_64-apple-darwin" "${repo_root}/.github/workflows/release.yml"
grep -q 'NIGHTWARD_VERSION="${version}"' "${repo_root}/.github/workflows/release.yml"
grep -q "dist/nightward_\\*.tar.gz" "${repo_root}/.github/workflows/release.yml"
grep -q "dist/nightward_\\*.zip" "${repo_root}/.github/workflows/release.yml"
if grep -q "path: dist/nightward_\\*" "${repo_root}/.github/workflows/release.yml"; then
  echo "expected release upload to exclude staging directories" >&2
  exit 1
fi

mkdir -p "${tmp}/target/release"
printf '#!/usr/bin/env bash\nprintf "0.1.0\\n"\n' >"${tmp}/target/release/nightward"
cp "${tmp}/target/release/nightward" "${tmp}/target/release/nw"
chmod 0755 "${tmp}/target/release/nightward" "${tmp}/target/release/nw"
(cd "${tmp}" && VERSION=0.1.0 "${repo_root}/scripts/release-snapshot-rust.sh" >/dev/null)
archive="$(find "${tmp}/dist" -name 'nightward_0.1.0_*_*.tar.gz' -print -quit)"
tar -tzf "${archive}" | grep -Fx "nightward" >/dev/null
tar -tzf "${archive}" | grep -Fx "nw" >/dev/null
if tar -tzf "${archive}" | grep -q '^nightward_0\.1\.0_'; then
  echo "expected release archive to contain root binaries, not a wrapper directory" >&2
  exit 1
fi

echo "release script fixture tests passed."
