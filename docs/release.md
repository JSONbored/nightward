# Release Process

Nightward releases are human-gated. Do not create a tag until the release candidate PR is reviewed and CI is green.

## Pre-Release Checklist

1. Confirm `main` is clean and up to date.
2. Run `make verify`.
3. Run `make release-snapshot`.
4. Confirm `renovate-config-validator renovate.json` passes.
5. Confirm `npm audit --audit-level=moderate` passes in both `integrations/raycast` and `packages/npm`.
6. Review `CHANGELOG.md` for user-facing changes, security notes, and breaking pre-1.0 behavior.
7. Confirm the OpenSSF evidence doc is current.

## Tagging

Use strict SemVer tags:

```sh
git tag -s v0.1.0 -m "chore(release): 0.1.0"
git push origin v0.1.0
```

Unsigned tags should not be used for public releases.

## GitHub Release

The release workflow runs GoReleaser from strict `vX.Y.Z` tags. It builds archives, creates checksums, generates SBOMs, signs `checksums.txt` with Cosign, and publishes the GitHub Release.

Verify a release locally with:

```sh
cosign verify-blob \
  --certificate-identity-regexp 'https://github.com/JSONbored/nightward/.github/workflows/release.yml@refs/tags/v.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --signature checksums.txt.sig \
  checksums.txt
```

Then verify an archive:

```sh
sha256sum -c checksums.txt --ignore-missing
```

## NPM Publish

The npm package is release-gated and disabled by default. To publish it:

1. Configure npm trusted publishing or `NPM_TOKEN`.
2. Set repository variable `NPM_PUBLISH=true`.
3. Push a reviewed release tag.

The npm job stamps `packages/npm/package.json` to the tag version, tests the launcher, audits dependencies, dry-runs package contents, and publishes with provenance.

The npm package should remain a launcher for GitHub Release binaries. Do not add a `postinstall` downloader or a second implementation of Nightward.
