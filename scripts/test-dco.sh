#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "${tmp_dir}"' EXIT

git -C "${tmp_dir}" init -q
git -C "${tmp_dir}" config user.name "Nightward Test"
git -C "${tmp_dir}" config user.email "nightward@example.test"
git -C "${tmp_dir}" config commit.gpgsign false

printf 'base\n' >"${tmp_dir}/file.txt"
git -C "${tmp_dir}" add file.txt
git -C "${tmp_dir}" commit -q -m "test: base"
base="$(git -C "${tmp_dir}" rev-parse HEAD)"

printf 'signed\n' >>"${tmp_dir}/file.txt"
git -C "${tmp_dir}" add file.txt
git -C "${tmp_dir}" commit -q -s -m "test: signed change"
(cd "${tmp_dir}" && "${repo_root}/scripts/check-dco.sh" "${base}" HEAD >/dev/null)

printf 'unsigned\n' >>"${tmp_dir}/file.txt"
git -C "${tmp_dir}" add file.txt
git -C "${tmp_dir}" commit -q -m "test: unsigned change"
if (cd "${tmp_dir}" && "${repo_root}/scripts/check-dco.sh" "${base}" HEAD >/dev/null 2>&1); then
  echo "DCO test expected unsigned commit to fail" >&2
  exit 1
fi

echo "DCO fixture tests passed."
