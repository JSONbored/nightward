#!/usr/bin/env bash
set -euo pipefail

tag="${1:?release tag required}"
repo="${GITHUB_REPOSITORY:-JSONbored/nightward}"
if [[ ! "${tag}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "invalid release tag: ${tag}" >&2
  exit 1
fi
version="${tag#v}"
asset="nightward_${version}_linux_amd64.tar.gz"
sbom="nightward_${version}_linux_amd64.sbom.json"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "${tmp_dir}"' EXIT

gh release download "${tag}" \
  --repo "${repo}" \
  --pattern checksums.txt \
  --pattern checksums.txt.sigstore.json \
  --pattern "${asset}" \
  --pattern "${sbom}" \
  --dir "${tmp_dir}"

cd "${tmp_dir}"
test -s "${sbom}"
cosign verify-blob \
  --bundle checksums.txt.sigstore.json \
  --certificate-identity-regexp "https://github.com/${repo}/.github/workflows/release.yml@refs/tags/v[0-9]+\\.[0-9]+\\.[0-9]+$" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt
sha256sum -c checksums.txt --ignore-missing
mkdir -p extracted
tar -xzf "${asset}" -C extracted

cd extracted
./nightward --version | grep -Fx "${version}"
./nw --version | grep -Fx "${version}"

echo "release archive smoke passed for ${repo}@${tag}"
