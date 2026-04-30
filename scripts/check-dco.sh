#!/usr/bin/env bash
set -euo pipefail

base="${1:-${BASE_SHA:-origin/main}}"
head="${2:-${HEAD_SHA:-HEAD}}"

commits="$(git rev-list --no-merges "${base}..${head}")"
missing=()
while IFS= read -r commit; do
  [[ -n "${commit}" ]] || continue
  if ! git log -1 --format=%B "${commit}" | grep -Eiq '^Signed-off-by: .+ <.+@.+>$'; then
    missing+=("${commit}")
  fi
done <<<"${commits}"

if (( ${#missing[@]} > 0 )); then
  echo "DCO sign-off missing from commits:" >&2
  printf '  %s\n' "${missing[@]}" >&2
  echo "Amend each commit with: git commit --amend -s --no-edit" >&2
  exit 1
fi

echo "DCO sign-off check passed."
