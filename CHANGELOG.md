# Changelog

Nightward uses Conventional Commit-style PR titles so release notes stay readable.

Formal changelog entries begin with the first tagged release.

## Unreleased

- Collapsed redundant `mcp_server_review` noise when stronger server-specific findings already exist.
- Added grouped fix-plan review items for repeated package-pin remediation.
- Added compact MCP policy output for AI-client status checks.
- Improved CLI/Raycast provider-token redaction and scoped Raycast fix-plan actions.
- Replaced pre-1.0 deterministic SHA-1 IDs with SHA-256-backed IDs while preserving the short ID shape.
- Added OpenSSF Silver-ready governance, threat model, DCO, coverage, and release snapshot hardening.
- Added a release-gated npm launcher package that downloads GitHub Release binaries on first run without `postinstall`.
- Added VitePress documentation site scaffolding and GitHub Pages deployment workflow.
- Hardened release publishing around GitHub release smoke checks and npm trusted publishing.
- Added rules list/explain commands and local static HTML report generation.
