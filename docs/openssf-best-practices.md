# OpenSSF Best Practices Evidence

Nightward tracks OpenSSF Best Practices evidence here so the badge entry can be updated without hunting through the repository.

Project badge: <https://www.bestpractices.dev/projects/12713>

## Passing Evidence

- Project website and repository: <https://github.com/JSONbored/nightward>
- Contribution process: <https://github.com/JSONbored/nightward/blob/main/CONTRIBUTING.md>
- Contribution requirements: <https://github.com/JSONbored/nightward/blob/main/CONTRIBUTING.md>
- License location: <https://github.com/JSONbored/nightward/blob/main/LICENSE>
- Documentation basics and interface docs: README, `site/`, plus `docs/action.md`, `docs/analysis.md`, `docs/remediation.md`, `docs/privacy-model.md`, and `docs/adapters.md`.
- Discussion and report archive: <https://github.com/JSONbored/nightward/issues>
- Vulnerability reporting: <https://github.com/JSONbored/nightward/blob/main/SECURITY.md>
- Build system: `Makefile`, `go.mod`, Raycast and npm launcher `package-lock.json` files, and GitHub Actions.
- Tests: `make test`, `make test-race`, `make test-junit`, `make coverage-check`, `make verify`.
- CI: `.github/workflows/ci.yml`, `.github/workflows/nightward-policy.yml`, `.github/workflows/scorecard.yml`.
- Static analysis: `go vet`, `staticcheck`, `gosec`, Trunk Check, CodeQL, Gitleaks, govulncheck, OSV, and Nightward SARIF.
- Dynamic analysis: automated tests, race tests, Raycast tests/build, and Go fuzz smoke tests.
- Secret scanning: Gitleaks in CI and `make gitleaks`.
- Release notes: <https://github.com/JSONbored/nightward/blob/main/CHANGELOG.md>
- Maintained status evidence after first release: reviewed PRs, CI-green `main`, Renovate dependency updates, issue response history, and signed release tags.

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
- Release readiness: `.goreleaser.yml`, `make release-snapshot`, signed checksum config, SBOM config, release workflow, release smoke, and trusted-publishing-only npm launcher package.
- Website/docs readiness: `site/` VitePress source and `.github/workflows/pages.yml`.
- Distribution plan: <https://github.com/JSONbored/nightward/blob/main/docs/distribution.md>

## Gold-Oriented Backlog

- SLSA provenance/attestations after the first signed release.
- Reproducible-build comparison using `CGO_ENABLED=0`, `-trimpath`, pinned Go toolchain, and rebuild verification.
- Wider fuzz/property tests for MCP parsing, redaction, symlink traversal, huge files, malformed configs, and permission-denied paths.
- More maintainer depth and documented two-person review coverage once the project has additional maintainers.
- Package-manager ecosystem coverage beyond npm and GitHub Releases.

## Manual Badge Actions

These cannot be completed by repository files alone:

- Save the OpenSSF badge form while logged in as a maintainer.
- Keep GitHub branch protection/rulesets requiring reviewed PRs and status checks.
- Keep maintainer 2FA enabled.
- Let Scorecard `Maintained` age out after the repository is older than 90 days.
- Improve Scorecard `Code-Review` through future reviewed PR history.
- Create the first signed release only after the release candidate is reviewed.
- Configure npm trusted publishing before setting `NPM_PUBLISH=true`.
- Enable GitHub Pages for the repository after the site workflow is merged.
