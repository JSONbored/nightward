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
- Read-only Raycast extension for scan summaries, findings, redacted fix-plan export, and report-folder access.
- User-level nightly schedule generation.
- Trunk Check and Trunk Flaky Tests JUnit validation.

## Next

- More MCP config shapes for Codex, Claude Code, and editor integrations.
- Richer TUI actions with real clipboard/export/open-doc behavior behind explicit keypresses.
- Raycast screenshots, store metadata, and manual development-mode smoke after the first release candidate.
- Golden SARIF snapshots and broader no-write tests.
- Validate GoReleaser on the first release candidate tag.
- Screenshot and GIF assets for the README.

## Later

- Encrypted local snapshots.
- Cross-machine diff.
- Private dotfiles integration.
- Docker/Unraid dashboard.
- Restore workflow only after snapshot, preview, rollback, and secret-safety controls exist.

## Non-Goals For V1

- Telemetry.
- Cloud dashboard.
- Live sync.
- Secret copying.
- Automatic config mutation.
- Git push automation.
