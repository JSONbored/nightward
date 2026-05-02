# Changelog

GitHub Releases are the canonical public release notes for Nightward. They are
generated with GitHub-native release notes through GoReleaser and organized by
the label categories in `.github/release.yml`.

This file is a lightweight curated index for shipped releases, superseded
release attempts, and high-signal unreleased work. Pull request titles should
continue to use Conventional Commit style so generated notes stay readable.

## Unreleased

- Collapsed redundant `mcp_server_review` noise when stronger server-specific findings already exist.
- Added grouped fix-plan review items for repeated package-pin remediation.
- Added compact MCP policy output for AI-client status checks.
- Improved CLI/Raycast provider-token redaction and scoped Raycast fix-plan actions.
- Replaced pre-1.0 deterministic SHA-1 IDs with SHA-256-backed IDs while preserving the short ID shape.
- Added OpenSSF Silver-ready governance, threat model, DCO, coverage, and release snapshot hardening.
- Added a release-gated npm launcher package that downloads GitHub Release binaries on first run without `postinstall`.
- Added VitePress documentation site scaffolding and GitHub Pages deployment workflow.
- Added rules list/explain commands and local static HTML report generation.
- Improved report history, website install UX, Raycast polish, and
  release-status documentation after the first stable release.

## v0.1.4

First stable Nightward release.

- Added the `nightward` and `nw` Go CLIs for local-first AI agent state, MCP
  config, and dotfiles backup-safety audits.
- Added MCP security findings for unpinned package execution, sensitive
  env/header references, local endpoints, broad filesystem access, local
  credential paths, parse failures, symlinked config, and unknown server shapes.
- Added redacted JSON scan output, policy checks, SARIF output, Trunk
  integration metadata, plan-only remediation, and static HTML reports.
- Added `nw` alias support, TUI review flows, and read-only Raycast extension
  surfaces.
- Added explicit local/online-capable provider execution support with opt-in
  provider gates.
- Added release-gated GitHub artifacts with checksums, SBOMs, Cosign-signed
  checksum bundles, release smoke checks, and npm trusted-publishing support for
  `@jsonbored/nightward`.
- Added OpenSSF-oriented governance, security policy, threat model, DCO, CodeQL,
  Scorecard, coverage, and release snapshot gates.

## v0.1.1-v0.1.3

Superseded prerelease attempts.

- `v0.1.1` and `v0.1.2` were superseded before the final npm/install
  verification path was complete.
- `v0.1.3` proved GitHub/npm publishing and provenance, but the npm launcher
  symlink smoke gap was fixed in `v0.1.4`.
- Use `v0.1.4` or newer.
