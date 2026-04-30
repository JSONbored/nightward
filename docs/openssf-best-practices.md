# OpenSSF Best Practices Evidence

Nightward tracks OpenSSF Best Practices evidence here so the badge entry can be updated without hunting through the repository.

Project badge: <https://www.bestpractices.dev/projects/12713>

## Passing Evidence

- Project website and repository: <https://github.com/JSONbored/nightward>
- Contribution process: <https://github.com/JSONbored/nightward/blob/main/CONTRIBUTING.md>
- Contribution requirements: <https://github.com/JSONbored/nightward/blob/main/CONTRIBUTING.md>
- License location: <https://github.com/JSONbored/nightward/blob/main/LICENSE>
- Documentation basics and interface docs: README plus `docs/action.md`, `docs/analysis.md`, `docs/remediation.md`, `docs/privacy-model.md`, and `docs/adapters.md`.
- Discussion and report archive: <https://github.com/JSONbored/nightward/issues>
- Vulnerability reporting: <https://github.com/JSONbored/nightward/blob/main/SECURITY.md>
- Build system: `Makefile`, `go.mod`, Raycast `package-lock.json`, and GitHub Actions.
- Tests: `make test`, `make test-race`, `make test-junit`, `make coverage-check`, `make verify`.
- CI: `.github/workflows/ci.yml`, `.github/workflows/nightward-policy.yml`, `.github/workflows/scorecard.yml`.
- Static analysis: `go vet`, `staticcheck`, `gosec`, Trunk Check, CodeQL, Gitleaks, govulncheck, OSV, and Nightward SARIF.
- Dynamic analysis: automated tests, race tests, Raycast tests/build, and Go fuzz smoke tests.
- Secret scanning: Gitleaks in CI and `make gitleaks`.

## N/A Crypto Fields

Nightward does not implement cryptographic protocols, encryption, password storage, key exchange, credential verification, cryptographic signing, or cryptographic key generation.

The release pipeline uses external tools for signing release checksums and SBOM generation. Nightward runtime does not provide cryptographic security mechanisms.

## Silver-Ready Evidence

- Governance: <https://github.com/JSONbored/nightward/blob/main/GOVERNANCE.md>
- Maintainers and access continuity: <https://github.com/JSONbored/nightward/blob/main/MAINTAINERS.md>
- Code of conduct: <https://github.com/JSONbored/nightward/blob/main/CODE_OF_CONDUCT.md>
- Roadmap: <https://github.com/JSONbored/nightward/blob/main/ROADMAP.md>
- Architecture/security model: `docs/privacy-model.md`, `docs/threat-model.md`, `docs/ci-security.md`, and `docs/remediation.md`.
- Dependency maintenance: <https://github.com/JSONbored/nightward/blob/main/docs/dependency-maintenance.md>
- DCO: `CONTRIBUTING.md` and the CI DCO sign-off job.
- Release readiness: `.goreleaser.yml`, `make release-snapshot`, signed checksum config, SBOM config, and release workflow.

## Manual Badge Actions

These cannot be completed by repository files alone:

- Save the OpenSSF badge form while logged in as a maintainer.
- Keep GitHub branch protection/rulesets requiring reviewed PRs and status checks.
- Keep maintainer 2FA enabled.
- Let Scorecard `Maintained` age out after the repository is older than 90 days.
- Improve Scorecard `Code-Review` through future reviewed PR history.
- Create the first signed release only after the release candidate is reviewed.
