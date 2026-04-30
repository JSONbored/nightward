# CI And Security Notes

Nightward's CI is meant to prove the project is serious about the same safety posture it recommends to users.

## Workflows

- `ci.yml`: Go tests, Gitleaks, govulncheck, and OSV dependency scanning.
- `nightward-policy.yml`: generates Nightward SARIF from a fixture home and uploads it to GitHub code scanning.
- `scorecard.yml`: runs OpenSSF Scorecard and uploads SARIF.

## Action Policy

- Pin third-party actions by full commit SHA.
- Keep the upstream tag in a nearby comment for maintainability.
- Use least-privilege workflow and job permissions.
- Prefer read-only `contents: read` unless a job needs SARIF upload or OIDC.

## Release Hardening Backlog

- Add GoReleaser after the initial CI workflows are stable.
- Add Sigstore/Cosign signing for release artifacts.
- Add provenance/SBOM output.
- Defer Homebrew tap automation until the first tagged release.
