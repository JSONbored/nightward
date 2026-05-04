#!/usr/bin/env bash
set -euo pipefail

version="${VERSION:-$(target/release/nightward --version)}"
os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "${arch}" in
  x86_64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "unsupported architecture: ${arch}" >&2; exit 2 ;;
esac

dist_dir="dist"
work_dir="${dist_dir}/nightward_${version}_${os}_${arch}"
archive="${work_dir}.tar.gz"

rm -rf "${dist_dir}"
mkdir -p "${work_dir}"
cp target/release/nightward target/release/nw "${work_dir}/"
chmod 0755 "${work_dir}/nightward" "${work_dir}/nw"

(cd "${work_dir}" && tar -czf "../$(basename "${archive}")" nightward nw)
(cd "${dist_dir}" && shasum -a 256 "$(basename "${archive}")" > checksums.txt)

cat > "${dist_dir}/nightward_${version}_${os}_${arch}.sbom.json" <<JSON
{
  "schema_version": 1,
  "generator": "nightward release-snapshot-rust",
  "version": "${version}",
  "target": "${os}-${arch}",
  "artifacts": ["$(basename "${archive}")"]
}
JSON

echo "created ${archive}"
