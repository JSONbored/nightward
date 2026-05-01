# Roadmap

Nightward is intentionally staged. The scanner and policy model need to stay trustworthy before backup or restore becomes live.

## Now

- Local inventory for Codex, Claude/Claude Code, Cursor, Windsurf, VS Code, Raycast, and generic MCP configs.
- Redacted JSON scan and doctor output.
- Read-only backup dry-run plans.
- Read-only remediation plans for MCP/security findings.
- Redacted fix previews for parseable MCP config changes.
- Optional `.nightward.yml` policy config with reason-required ignores.
- Read-only snapshot plan and diff commands.
- SARIF and policy output for CI.
- GitHub Action wrapper for repository policy checks.
- Release-gated npm launcher package with checksum verification and no postinstall.
- Read-only Raycast extension for scan summaries, menu-bar status, findings, provider controls, redacted fix-plan export, and report-folder access.
- User-level nightly schedule generation.
- Trunk Check and Trunk Flaky Tests JUnit validation.
- Explicit provider execution for local `gitleaks`, `trufflehog`, and `semgrep`, plus online-gated `trivy`, `osv-scanner`, and `socket`.
- Static HTML reports, report diffs, report history, generated reference docs, sample reports, and fixture-only demo assets.
- Rules and adapter list/explain/template commands for contributors.

## Next

- More MCP config shapes for Codex, Claude Code, and editor integrations.
- Raycast screenshots, store metadata, and manual development-mode smoke evidence.
- Golden SARIF snapshots and broader no-write tests.
- Bubble Tea/Bubbles TUI upgrade with list, table, viewport, textinput, help, filter modal, command palette, report history, and mouse-wheel support.
- Deeper provider normalization, provider-specific fixtures, and clearer skip/timeout/output-cap reporting across CLI, TUI, Raycast, SARIF, policy, and HTML.
- Fuzz coverage for MCP JSON/TOML/YAML parsing, URL/header redaction, symlink traversal, huge files, and malformed configs.
- Add SLSA provenance/attestation after signed checksum and npm provenance flow are stable.
- Report-history comparison as a first-class workflow across TUI, Raycast, CLI, and HTML.
- Richer local HTML review artifacts with search/filtering, evidence grouping, provider-warning summaries, policy status, and report-to-report comparisons.
- Screenshot and GIF assets for the README and Raycast store listing.
- Generated public JSON schemas for scan, analysis, policy badge, report diff/history, provider status, rules, and adapters.
- Homebrew tap after the first release proves stable.

## Later

- Local read-only report browser before any Docker-first dashboard positioning.
- Encrypted local snapshots.
- Cross-machine diff.
- Private dotfiles integration.
- Docker/Unraid dashboard only after report browsing is useful outside the host shell.
- Nix, Scoop, WinGet, mise, and aqua packages.
- Restore workflow only after snapshot, preview, rollback, and secret-safety controls exist.

## Non-Goals For V1

- Telemetry.
- Cloud dashboard.
- Live sync.
- Secret copying.
- Automatic config mutation.
- Git push automation.
