# Distribution Plan

Nightward distribution should optimize for trust first, then convenience.

## Order

1. GitHub Releases with signed checksums and SBOMs. Shipped in `v0.1.4`.
2. Scoped npm launcher `@jsonbored/nightward` with trusted publishing and provenance. Shipped in `v0.1.4`.
3. Source builds with `make install-local`. Development-only.
4. Trunk plugin import from a release tag. Shipped.
5. GitHub Action release tags. Shipped.
6. Homebrew formula generation from signed release checksums. Scaffolded.
7. Homebrew tap publication.
8. Nix flake/package.
9. Scoop and WinGet.
10. mise and aqua.

Docker is deferred until Nightward has a useful local report browser. A container is not a good default for safely scanning a user's HOME directory.

## Homebrew Tap Support

`scripts/generate-homebrew-formula.mjs` generates a tap-ready formula from the signed GitHub Release checksum file. The formula points at the existing `nightward_<version>_<os>_<arch>.tar.gz` archives, installs both `nightward` and `nw`, and tests both command names with `--version`.

The release verifier runs the generator after Cosign verifies `checksums.txt.sigstore.json` and `sha256sum` validates the downloaded archive. That keeps Homebrew support tied to the release artifact shape instead of creating a second packaging implementation. A dedicated tap repository is still a separate publication step.

## NPM Posture

The npm package is `@jsonbored/nightward`. It is published through npm trusted publishing and must remain a launcher:

- no `postinstall`
- no bundled second implementation
- no long-lived npm token
- trusted publishing with provenance
- checksum verification before executing downloaded release archives

## Ongoing Release Checklist

1. Merge through reviewed PR when a non-author reviewer exists, or document a solo-maintainer admin bypass.
2. Confirm branch and tag protection.
3. Confirm npm trusted publishing for `.github/workflows/release.yml`.
4. Run local verification.
5. Create a signed SemVer tag.
6. Verify GitHub release artifacts.
7. Generate/check the Homebrew formula from signed release checksums.
8. Verify npm metadata and launcher behavior with `scripts/verify-npm-release.sh`.
9. Update OpenSSF badge evidence.
