# Release Process

Nightward releases are human-gated. Do not create a tag until the release candidate PR is reviewed and CI is green.

## Pre-Release Checklist

1. Confirm `main` is clean and up to date.
2. Run `make verify`.
3. Run `make release-snapshot`.
4. Confirm `renovate-config-validator renovate.json` passes.
5. Confirm `npm audit --audit-level=moderate` passes in both `integrations/raycast` and `packages/npm`.
6. Confirm `npm pack --dry-run` passes in `packages/npm`.
7. Confirm `npm audit signatures` where dependencies exist; the npm launcher currently has no runtime dependencies, so this may report that there is nothing to audit.
8. Review `CHANGELOG.md` for user-facing changes, security notes, and breaking pre-1.0 behavior.
9. Confirm the OpenSSF evidence doc is current.
10. After publish, run the release and npm verification scripts for the released version.

## Release Protection

Use these repository settings before the first public release:

- Require pull request review before merging to `main`.
- Require CI, CodeQL, Nightward Policy, Scorecard, DCO, and Release snapshot checks.
- Protect release tags from force-push and deletion.
- Require signed tags for releases.
- Use a protected `npm-publish` environment with at least one reviewer.
- Configure npm trusted publishing for package `@jsonbored/nightward`, repository `JSONbored/nightward`, workflow `.github/workflows/release.yml`.

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
  --new-bundle-format=false \
  --certificate-identity-regexp 'https://github.com/JSONbored/nightward/.github/workflows/release.yml@refs/tags/v.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate checksums.txt.pem \
  --signature checksums.txt.sig \
  checksums.txt
```

Then verify an archive:

```sh
sha256sum -c checksums.txt --ignore-missing
```

## NPM Publish

The npm package is release-gated, trusted-publishing only, and disabled by default. To publish it:

1. Configure npm trusted publishing for package `@jsonbored/nightward`.
2. Set repository variable `NPM_PUBLISH=true`.
3. Push a reviewed, signed release tag.

The npm job stamps `packages/npm/package.json` to the tag version, tests the launcher, audits dependencies, packs the tarball, installs that tarball into a temporary prefix, smokes both `nightward` and `nw`, dry-runs package contents, and publishes with provenance.

The npm package should remain a launcher for GitHub Release binaries. Do not add a `postinstall` downloader or a second implementation of Nightward.

After publish, verify:

```sh
npm view @jsonbored/nightward version dist.integrity repository
npm view @jsonbored/nightward --json | jq '.dist'
npx @jsonbored/nightward --version
npm install -g @jsonbored/nightward
nw doctor --json
```

If a bad npm package is published, deprecate that version with a clear reason and publish a fixed patch release. Do not unpublish unless npm policy and the severity of the issue require it.

## Release Smoke

The release workflow smokes the published GitHub archive before npm publish:

```sh
bash scripts/smoke-release-archive.sh v0.1.0
```

The npm job then installs the packed npm tarball and runs both command names before publishing.

After npm publish, verify package metadata and launcher install smoke:

```sh
bash scripts/verify-npm-release.sh 0.1.0
```
