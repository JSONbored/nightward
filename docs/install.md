# Install And Release Channels

Nightward should be easy to try without weakening its security posture. The preferred release model is signed GitHub Release artifacts first, then convenience installers that point back to those artifacts.

## Recommended Install

```sh
npx @jsonbored/nightward scan
npm install -g @jsonbored/nightward
nw
```

`npx @jsonbored/nightward scan` is the fastest read-only trial. `npm install -g @jsonbored/nightward` installs the same release-backed launcher for repeated CLI and TUI use. The TUI and CLI are not separate packages: `nw` opens the TUI, and `nw scan`, `nw policy`, `nw report`, and the other subcommands use the same installed binary.

## Local Source Install

```sh
make install-local
```

This builds `nightward` and `nw` from the local checkout into `~/.local/bin` by default. Use this for development or branch comparison, not as the recommended end-user install path.

## GitHub Releases

GitHub Releases are the canonical binary distribution channel.

Release artifacts include:

- `nightward` and `nw` Rust binaries for macOS, Linux, and Windows x64.
- `checksums.txt`.
- `checksums.txt.sigstore.json` from Cosign keyless signing.
- SBOM files for release archives.

Users who want the strongest supply-chain verification should download from GitHub Releases and verify the signed checksum file before installing.

## NPM

The npm registry rejected the unscoped `nightward` package name as too similar to an existing package, so Nightward publishes the scoped package `@jsonbored/nightward`. The installed binaries are still `nightward` and `nw`.

The package is a thin launcher:

- no `postinstall` script
- no bundled Node implementation of Nightward
- downloads the matching GitHub Release archive on first run
- verifies the archive SHA-256 from `checksums.txt`
- rejects absolute, parent-directory, symlink, duplicate, or unexpected archive entries before extraction
- optionally requires Cosign verification of `checksums.txt.sigstore.json` when `NIGHTWARD_NPM_REQUIRE_SIGSTORE=1` is set
- caches extracted `nightward` and `nw` binaries locally
Windows ARM64 remains deferred until the release matrix includes a validated Windows ARM64 Rust build.

Example:

```sh
npx @jsonbored/nightward scan
npm install -g @jsonbored/nightward
nw scan --json
```

Publishing is release-gated. The release workflow publishes to npm only when `NPM_PUBLISH=true` is configured for a release tag and npm trusted publishing is configured for package `@jsonbored/nightward`, repository `JSONbored/nightward`, and workflow `.github/workflows/release.yml`.

The package should not use a long-lived npm token. It should publish through GitHub OIDC/trusted publishing with provenance.

## Deferred Channels

These are useful, but should wait until signed GitHub Release artifacts prove stable across patch releases:

- Homebrew tap
- Nix package
- mise/aqua registry entries
- Docker image for report browsing
