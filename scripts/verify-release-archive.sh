#!/usr/bin/env bash
set -euo pipefail

tag="${1:?release tag required}"
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
repo="${GITHUB_REPOSITORY:-JSONbored/nightward}"
if [[ ! "${tag}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "invalid release tag: ${tag}" >&2
  exit 1
fi
version="${tag#v}"
case "$(uname -s)" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  MINGW* | MSYS* | CYGWIN*) os="windows" ;;
  *)
    echo "unsupported release verification OS: $(uname -s)" >&2
    exit 1
    ;;
esac
case "$(uname -m)" in
  x86_64 | amd64) arch="amd64" ;;
  arm64 | aarch64) arch="arm64" ;;
  *)
    echo "unsupported release verification architecture: $(uname -m)" >&2
    exit 1
    ;;
esac
if [[ "${os}" == "windows" ]]; then
  asset="nightward_${version}_${os}_${arch}.zip"
else
  asset="nightward_${version}_${os}_${arch}.tar.gz"
fi
sbom="nightward_${version}_${os}_${arch}.sbom.json"
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
node "${repo_root}/scripts/generate-homebrew-formula.mjs" \
  --version "${version}" \
  --repo "${repo}" \
  --checksums checksums.txt \
  --output "${tmp_dir}/homebrew/nightward.rb" >/dev/null
grep -q 'bin.install "nightward", "nw"' "${tmp_dir}/homebrew/nightward.rb"
grep -q '#{bin}/nightward --version' "${tmp_dir}/homebrew/nightward.rb"
grep -q '#{bin}/nw --version' "${tmp_dir}/homebrew/nightward.rb"
mkdir -p extracted
if [[ "${asset}" == *.zip ]]; then
  unzip -q "${asset}" -d extracted
  bin_ext=".exe"
else
  tar -xzf "${asset}" -C extracted
  bin_ext=""
fi

cd extracted
./nightward"${bin_ext}" --version | grep -Fx "${version}"
./nw"${bin_ext}" --version | grep -Fx "${version}"

echo "release archive verification passed for ${repo}@${tag}"
