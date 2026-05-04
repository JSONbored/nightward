#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
validator="${repo_root}/scripts/validate-action-path.sh"

"${validator}" output "nightward.sarif"
"${validator}" output "reports/nightward.sarif"
"${validator}" output "-"
"${validator}" config ".nightward.yml"

for value in "/tmp/nightward.sarif" "../nightward.sarif" "reports/../nightward.sarif" $'bad\npath' 'bad\path'; do
  if "${validator}" output "${value}" >/dev/null 2>&1; then
    echo "expected unsafe action path to fail: ${value}" >&2
    exit 1
  fi
done

tmp="$(mktemp -d)"
trap 'rm -rf "${tmp}"' EXIT
mkdir -p "${tmp}/workspace" "${tmp}/outside"
GITHUB_WORKSPACE="${tmp}/workspace" "${validator}" output "safe/nightward.sarif"
ln -s "${tmp}/outside/leak.sarif" "${tmp}/workspace/nightward.sarif"
if GITHUB_WORKSPACE="${tmp}/workspace" "${validator}" output "nightward.sarif" >/dev/null 2>&1; then
  echo "expected symlinked output file to fail" >&2
  exit 1
fi
ln -s "${tmp}/outside" "${tmp}/workspace/reports"
if GITHUB_WORKSPACE="${tmp}/workspace" "${validator}" output "reports/nightward.sarif" >/dev/null 2>&1; then
  echo "expected symlinked output parent to fail" >&2
  exit 1
fi

echo "action path fixture tests passed."
