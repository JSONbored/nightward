# CI And Security Notes

Nightward's CI is meant to prove the project is serious about the same safety posture it recommends to users.

## Workflows

- `ci.yml`: Go tests, race tests, coverage gate, `go vet`, `staticcheck`, `gosec`, fuzz smoke tests, JUnit reports, local JUnit shape validation, gated Trunk Flaky Tests uploads, explicit Trunk Check CLI execution, Raycast extension tests/build/audit, Gitleaks, govulncheck, OSV dependency scanning, DCO checking, and GoReleaser snapshot validation.
- `nightward-policy.yml`: generates workspace Nightward SARIF and uploads it to GitHub code scanning without scanning synthetic risky fixture homes.
- `plugin.yaml`: defines Trunk Check linters for workspace policy and analysis SARIF once release tags are available.
- `scorecard.yml`: runs OpenSSF Scorecard on PRs, `main`, branch-protection changes, and a weekly schedule. PR runs do not publish results or upload SARIF; `main` and scheduled runs upload SARIF.
- `release.yml`: publishes signed GoReleaser artifacts from strict `vX.Y.Z` tags.
- `renovate.json`: manages Go modules, Raycast npm packages, pinned GitHub Actions, local tool pins, and release tooling updates.

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
- Use repo-controlled `make gitleaks` and `make govulncheck` targets in CI so local and remote behavior match.
- Install Trunk in CI from a pinned release archive with a checked SHA-256 instead of a moving launcher URL.
- Keep Trunk Flaky Tests secrets scoped to the detection/upload steps only.
- Keep composite action output/config paths relative to `GITHUB_WORKSPACE`; reject absolute paths, parent traversal, and newlines.
- Require DCO sign-offs on pull request commits.

## Trunk Plugin Notes

The in-repo plugin exposes:

- `nightward-policy`: runs `nw policy sarif --workspace ${workspace} --output -`
- `nightward-analyze`: runs `nw policy sarif --workspace ${workspace} --include-analysis --output -`

Users should import a pinned Nightward tag, not a moving branch:

```sh
trunk plugins add --id nightward https://github.com/JSONbored/nightward v0.1.0
trunk check enable nightward-policy
```

## Release Hardening Backlog

- Validate GoReleaser on the first release candidate tag.
- Add provenance once release artifact flow is stable.
- Defer Homebrew tap automation until the first tagged release.
