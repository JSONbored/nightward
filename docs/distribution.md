# Distribution Plan

Nightward distribution should optimize for trust first, then convenience.

## Order

1. GitHub Releases with signed checksums and SBOMs. Shipped.
2. Scoped npm launcher `@jsonbored/nightward` with trusted publishing and provenance. Shipped.
3. Source builds with `make install-local`. Development-only.
4. Trunk plugin import from a release tag. Shipped.
5. GitHub Action release tags. Shipped.
6. Homebrew tap.
7. Nix flake/package.
8. Scoop and WinGet.
9. mise and aqua.

Docker is deferred until Nightward has a useful local report browser. A container is not a good default for safely scanning a user's HOME directory.

## Homebrew Tap Plan

Homebrew is straightforward once the release archive layout stays stable. Add a dedicated tap repository, then generate a formula from the signed GitHub Release archive and checksum data. The formula should install both `nightward` and `nw`, include a lightweight `nightward --version` test, and avoid becoming a separate packaging implementation.

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
7. Verify npm metadata and launcher behavior with `scripts/verify-npm-release.sh`.
8. Update OpenSSF badge evidence.
