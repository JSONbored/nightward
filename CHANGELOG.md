# Changelog

GitHub Releases are the canonical public release notes for Nightward. They are
generated with GitHub-native release notes and organized by
the label categories in `.github/release.yml`.

This file is a lightweight curated index for shipped releases, superseded
release attempts, and high-signal unreleased work. Pull request titles should
continue to use Conventional Commit style so generated notes stay readable.

## Unreleased

## v0.1.6

Release repair and workflow hardening after the protected `v0.1.5` tag could
not be reused.

- Added parser fuzz harnesses and regression coverage for MCP JSON/TOML/YAML
  parsing, redaction, symlink traversal, huge files, and malformed configs.
- Added first-class report-history comparison across CLI data, TUI, Raycast,
  static HTML reports, tests, and docs.
- Tightened provider normalization for `gitleaks`, `trufflehog`, `semgrep`,
  `trivy`, `osv-scanner`, and `socket`, including skip, timeout, output-cap,
  provider-warning, policy, SARIF, TUI, Raycast, and HTML behavior.
- Hardened release publishing with signed-tag verification, Windows-compatible
  builds for non-TUI commands, scoped archive uploads, and Cosign-backed release
  verification before npm publication.

## v0.1.5

Superseded release attempt.

- The signed `v0.1.5` tag was created while repairing the release workflow, but
  the GitHub Release was removed before npm publication completed. The protected
  tag remains in the repository and should be skipped; use `v0.1.6` or newer.

## v0.1.4

First stable Nightward release.

- Added the `nightward` and `nw` CLIs for local-first AI agent state, MCP
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
  checksum bundles, release archive verification, and npm trusted-publishing support for
  `@jsonbored/nightward`.
- Added OpenSSF-oriented governance, security policy, threat model, DCO, CodeQL,
  Scorecard, coverage, and release snapshot gates.

## v0.1.1-v0.1.3

Superseded prerelease attempts.

- `v0.1.1` and `v0.1.2` were superseded before the final npm/install
  verification path was complete.
- `v0.1.3` proved GitHub/npm publishing and provenance, but the npm launcher
  symlink verification gap was fixed in `v0.1.4`.
- Use `v0.1.4` or newer.
