# Dependency Maintenance

Nightward uses Renovate instead of Dependabot.

## What Renovate Owns

- Go modules in `go.mod` and `go.sum`.
- Raycast extension packages in `integrations/raycast/package.json` and `package-lock.json`.
- NPM launcher package metadata in `packages/npm/package.json` and `package-lock.json`.
- GitHub Actions versions and pinned action digests.
- `gotestsum`, `gitleaks`, `govulncheck`, `gosec`, `staticcheck`, GoReleaser, and Syft pins in `Makefile`.
- GoReleaser binary version in `.github/workflows/release.yml`.
- Trunk plugin registry pin in `.trunk/trunk.yaml`.

## Policy

- Keep third-party GitHub Actions pinned by full commit SHA.
- Keep the readable upstream action tag in a nearby comment.
- Use Conventional Commit PR titles, with dependency updates under `chore(deps):`.
- Do not enable automerge by default.
- Let Renovate run `go mod tidy` after Go module updates.
- Keep Dependabot disabled unless there is a specific ecosystem gap Renovate cannot cover.

## Review Expectations

Dependency PRs should still be reviewed like code changes. Check:

- changed release notes for security or breaking changes
- workflow diffs for permission or publish-surface changes
- `go.sum` changes for unexpected transitive churn
- `integrations/raycast/package-lock.json` changes for unexpected runtime dependency additions
- `packages/npm/package-lock.json` changes for unexpected launcher dependency additions
- CI results for Go tests, Raycast extension tests/build, Trunk Check, Gitleaks, govulncheck, OSV, and Nightward SARIF
- checked SHA changes in scripts that install pinned CI tools such as Trunk
