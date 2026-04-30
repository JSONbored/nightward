# CI And Security Notes

Nightward's CI is meant to prove the project is serious about the same safety posture it recommends to users.

## Workflows

- `ci.yml`: Go tests, JUnit reports, local JUnit shape validation, gated Trunk Flaky Tests uploads, Trunk Check, Gitleaks, govulncheck, and OSV dependency scanning.
- `nightward-policy.yml`: generates Nightward SARIF from a fixture home and uploads it to GitHub code scanning.
- `scorecard.yml`: runs OpenSSF Scorecard and uploads SARIF.
- `release.yml`: publishes signed GoReleaser artifacts from strict `vX.Y.Z` tags.
- `renovate.json`: manages Go modules, pinned GitHub Actions, local tool pins, and release tooling updates.

## Action Policy

- Pin third-party actions by full commit SHA.
- Keep the upstream tag in a nearby comment for maintainability.
- Use least-privilege workflow and job permissions.
- Prefer read-only `contents: read` unless a job needs SARIF upload or OIDC.
- Keep Trunk Flaky Tests uploads gated on `TRUNK_ORG_URL_SLUG` and `TRUNK_API_TOKEN`.
- Never make flaky-test quarantining a default CI behavior.
- Use Renovate instead of Dependabot whenever possible.
- Keep dependency PRs reviewed; do not enable broad automerge by default.

## Release Hardening Backlog

- Validate GoReleaser on the first release candidate tag.
- Add provenance once release artifact flow is stable.
- Defer Homebrew tap automation until the first tagged release.
