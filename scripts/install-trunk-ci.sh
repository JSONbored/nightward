#!/usr/bin/env bash
set -euo pipefail

version="${TRUNK_VERSION:-1.25.0}"
install_dir="${TRUNK_INSTALL_DIR:-/usr/local/bin}"
platform="${TRUNK_PLATFORM:-linux}"
arch="${TRUNK_ARCH:-x86_64}"

if [[ "${platform}" != "linux" || "${arch}" != "x86_64" ]]; then
  echo "unsupported Trunk CI platform: ${platform}-${arch}" >&2
  exit 2
fi

expected_sha256="${TRUNK_LINUX_X86_64_SHA256:-3845ff76a70cebb10e61a267ff719ffdccfa3ef6d877d51870a2c62b79603ab9}"
url="https://trunk.io/releases/${version}/trunk-${version}-${platform}-${arch}.tar.gz"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "${tmp_dir}"' EXIT

curl -fsSLo "${tmp_dir}/trunk.tar.gz" "${url}"
if command -v sha256sum >/dev/null 2>&1; then
  actual_sha256="$(sha256sum "${tmp_dir}/trunk.tar.gz" | awk '{print $1}')"
else
  actual_sha256="$(shasum -a 256 "${tmp_dir}/trunk.tar.gz" | awk '{print $1}')"
fi
if [[ "${actual_sha256}" != "${expected_sha256}" ]]; then
  echo "Trunk checksum mismatch for ${url}" >&2
  echo "expected: ${expected_sha256}" >&2
  echo "actual:   ${actual_sha256}" >&2
  exit 1
fi

tar -xzf "${tmp_dir}/trunk.tar.gz" -C "${tmp_dir}"
install -m 0755 "${tmp_dir}/trunk" "${install_dir}/trunk"
