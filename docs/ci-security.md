# CI And Security Notes

Nightward's CI is meant to prove the project is serious about the same safety posture it recommends to users.

## Workflows

- `ci.yml`: Rust formatting, Clippy, tests, doc tests, coverage gate, explicit Trunk Check CLI execution, Raycast extension tests/build/audit, npm launcher tests/audit/package dry-run, Gitleaks, OSV dependency scanning, DCO checking, and Rust release snapshot validation.
- `nightward-policy.yml`: generates workspace Nightward SARIF, uploads a Nightward badge JSON artifact, and uploads SARIF to GitHub code scanning without scanning synthetic risky fixture homes.
- `nw policy badge`: writes a local JSON status artifact for dashboards or release evidence when a workflow explicitly requests it.
- `plugin.yaml`: defines Trunk Check linters for workspace policy and analysis SARIF once release tags are available.
- `scorecard.yml`: runs OpenSSF Scorecard on PRs, `main`, branch-protection changes, and a weekly schedule. PR runs do not publish results or upload SARIF; `main` and scheduled runs upload SARIF.
- `release.yml`: publishes signed Rust artifacts from strict `vX.Y.Z` tags, verifies published archives, and can publish the npm launcher only through trusted publishing when explicitly enabled.
- `pages.yml`: builds and deploys the VitePress documentation site from `site/` to GitHub Pages.
- `renovate.json`: manages Cargo dependencies, Raycast/npm packages, pinned GitHub Actions, local tool pins, and release tooling updates.

## Action Policy

- Pin third-party actions by full commit SHA.
- Keep the upstream tag in a nearby comment for maintainability.
- Use least-privilege workflow and job permissions.
- Prefer read-only `contents: read` unless a job needs SARIF upload or OIDC.
- Keep OpenSSF Scorecard publish permissions job-scoped; global `id-token: write` fails Scorecard workflow verification.
- Keep release publish permissions job-scoped; top-level workflow permissions should stay read-only unless every job truly needs write access.
- Keep Trunk Flaky Tests uploads gated on `TRUNK_ORG_URL_SLUG` and `TRUNK_API_TOKEN`.
- Never make flaky-test quarantining a default CI behavior.
- Use Renovate instead of Dependabot whenever possible.
- Keep dependency PRs reviewed; do not enable broad automerge by default.
- `main` branch protection requires two approving reviews and CODEOWNERS review. While Nightward has only one maintainer, maintainer merges require an explicit admin bypass and an issue/PR note explaining why normal review could not be satisfied.
- Use repo-controlled `make gitleaks`, `make cargo-audit`, and `make cargo-deny` targets locally so local and remote behavior stay aligned.
- Install Trunk in CI from a pinned release archive with a checked SHA-256 instead of a moving launcher URL.
- Keep Trunk Flaky Tests secrets scoped to the detection/upload steps only.
- Keep composite action output/config paths relative to `GITHUB_WORKSPACE`; reject absolute paths, parent traversal, and newlines.
- Require DCO sign-offs on pull request commits.
- Keep the npm package free of `postinstall`; publish only from reviewed tags with trusted publishing and provenance.
- Keep the `npm-publish` and `github-pages` environments protected where repository settings allow it.

## Trunk Plugin Notes

The in-repo plugin exposes:

- `nightward-policy`: runs `nw policy sarif --workspace ${workspace} --output -`
- `nightward-analyze`: runs `nw policy sarif --workspace ${workspace} --include-analysis --output -`

Users should import a pinned Nightward tag, not a moving branch:

```sh
trunk plugins add --id nightward https://github.com/JSONbored/nightward v0.1.11
trunk check enable nightward-policy
```

## Release Hardening Backlog

- Add SLSA provenance/attestations once release artifact flow is stable.
- Add Homebrew tap automation after the current release artifacts prove stable across patch releases.

## Local Test Suites

Use suite aliases before pushing:

```sh
make test-fast
make test-security
make test-ux
make test-release
make test-prepush
```

`make test-prepush` is the full local release gate. It should pass before pushing a release-sensitive branch.
