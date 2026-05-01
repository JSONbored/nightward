#!/usr/bin/env bash
set -euo pipefail

version="${1:?package version required, for example 0.1.0}"
package="${NPM_PACKAGE:-@jsonbored/nightward}"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "${tmp_dir}"' EXIT

if [[ ! "${version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "invalid npm package version: ${version}" >&2
  exit 1
fi

metadata="$(npm view "${package}@${version}" --json)"
node -e '
const metadata = JSON.parse(process.argv[1]);
if (metadata.name !== process.argv[2]) throw new Error(`unexpected package name ${metadata.name}`);
if (metadata.version !== process.argv[3]) throw new Error(`unexpected package version ${metadata.version}`);
if (!metadata.dist || !metadata.dist.integrity) throw new Error("missing dist.integrity");
if (!metadata.repository) throw new Error("missing repository metadata");
console.log(`${metadata.name}@${metadata.version} ${metadata.dist.integrity}`);
' "${metadata}" "${package}" "${version}"

tarball="$(npm pack "${package}@${version}" --silent --pack-destination "${tmp_dir}")"
prefix="${tmp_dir}/prefix"
npm install --global --prefix "${prefix}" "${tmp_dir}/${tarball}" --ignore-scripts --no-audit

PATH="${prefix}/bin:${PATH}" nightward --version | grep -Fx "${version}"
PATH="${prefix}/bin:${PATH}" nw --version | grep -Fx "${version}"

echo "npm release smoke passed for ${package}@${version}"
