# Install And Release Channels

Nightward should be easy to try without weakening its security posture. The preferred release model is signed GitHub Release artifacts first, then convenience installers that point back to those artifacts.

## Current Local Install

```sh
make install-local
```

This builds `nightward` and `nw` from the local checkout into `~/.local/bin` by default.

## GitHub Releases

After the first reviewed release tag, GitHub Releases are the canonical binary distribution channel.

Release artifacts include:

- `nightward` and `nw` binaries for macOS, Linux, and Windows.
- `checksums.txt`.
- `checksums.txt.sig` from Cosign.
- SBOM files for release archives.

Users who want the strongest supply-chain verification should download from GitHub Releases and verify the signed checksum file before installing.

## NPM

The `nightward` npm package name is available and this repo includes a release-gated package under `packages/npm`.

The package is a thin launcher:

- no `postinstall` script
- no bundled Node implementation of Nightward
- downloads the matching GitHub Release archive on first run
- verifies the archive SHA-256 from `checksums.txt`
- caches extracted `nightward` and `nw` binaries locally

Example after the first release is published:

```sh
npx nightward --help
npm install -g nightward
nw scan --json
```

Publishing is disabled by default. The release workflow publishes to npm only when `NPM_PUBLISH=true` is configured and npm credentials/trusted publishing are ready.

## Deferred Channels

These are useful, but should wait until the first signed GitHub Release proves stable:

- Homebrew tap
- Nix package
- mise/aqua registry entries
- Docker image for report browsing
