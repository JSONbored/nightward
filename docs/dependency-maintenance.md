# Dependency Maintenance

Nightward uses Renovate instead of Dependabot.

## What Renovate Owns

- Cargo workspace dependencies in `Cargo.toml` and `Cargo.lock`.
- Raycast extension packages in `integrations/raycast/package.json` and `package-lock.json`.
- NPM launcher package metadata in `packages/npm/package.json` and `package-lock.json`.
- VitePress documentation site packages in `site/package.json` and `package-lock.json`.
- GitHub Actions versions and pinned action digests.
- Rust toolchain pin in `rust-toolchain.toml`.
- Gitleaks and release helper behavior in `Makefile` and scripts.
- Trunk plugin registry pin in `.trunk/trunk.yaml`.

## Policy

- Keep third-party GitHub Actions pinned by full commit SHA.
- Keep the readable upstream action tag in a nearby comment.
- Use Conventional Commit PR titles, with dependency updates under `chore(deps):`.
- Do not enable automerge by default.
- Let Renovate update Cargo dependencies and keep `Cargo.lock` changes reviewed.
- Keep Dependabot disabled unless there is a specific ecosystem gap Renovate cannot cover.

## Review Expectations

Dependency PRs should still be reviewed like code changes. Check:

- changed release notes for security or breaking changes
- workflow diffs for permission or publish-surface changes
- `Cargo.lock` changes for unexpected transitive churn
- `integrations/raycast/package-lock.json` changes for unexpected runtime dependency additions
- `packages/npm/package-lock.json` changes for unexpected launcher dependency additions
- `site/package-lock.json` changes for unexpected docs runtime script additions
- CI results for Rust tests, Raycast extension tests/build, Trunk Check, Gitleaks, OSV, and Nightward SARIF
- checked SHA changes in scripts that install pinned CI tools such as Trunk
