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
8. Review the generated GitHub Release notes and the curated `CHANGELOG.md`
   index for user-facing changes, security notes, and breaking pre-1.0
   behavior.
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
git tag -s vX.Y.Z -m "chore(release): X.Y.Z"
git push origin vX.Y.Z
```

Unsigned tags should not be used for public releases.

## GitHub Release

The release workflow builds Rust binaries from strict `vX.Y.Z` tags. It packages archives, creates checksums, signs `checksums.txt` with Cosign, and publishes the GitHub Release.

GitHub Releases are the canonical public release notes. The release workflow delegates
changelog generation to GitHub-native generated release notes, organized by
`.github/release.yml` label categories. Keep release PR titles in Conventional
Commit style and apply release-note labels before tagging so the generated notes
are useful without manual editing.

`CHANGELOG.md` is a lightweight curated index. Update it for stable releases,
superseded release attempts, security notes, and important pre-1.0 compatibility
notes, but do not duplicate every generated release-note entry there.

Verify a release locally with:

```sh
cosign verify-blob \
  --bundle checksums.txt.sigstore.json \
  --certificate-identity-regexp 'https://github.com/JSONbored/nightward/.github/workflows/release.yml@refs/tags/v[0-9]+\.[0-9]+\.[0-9]+$' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
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

The npm job stamps `packages/npm/package.json` to the tag version, tests the launcher, audits dependencies, packs the tarball, dry-runs package contents, and publishes with provenance.

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

## Release Verification

The release workflow verifies the published GitHub archive before npm publish:

```sh
bash scripts/verify-release-archive.sh vX.Y.Z
```

That verifier also generates a Homebrew formula from the signed `checksums.txt` file and checks that the formula installs/tests both `nightward` and `nw`.

The npm job then installs the packed npm tarball and runs both command names before publishing.

After npm publish, verify package metadata and launcher install behavior:

```sh
bash scripts/verify-npm-release.sh X.Y.Z
```
