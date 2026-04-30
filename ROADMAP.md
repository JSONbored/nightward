# Roadmap

Nightward is intentionally staged. The scanner and policy model need to stay trustworthy before backup or restore becomes live.

## Now

- Local inventory for Codex, Claude/Claude Code, Cursor, Windsurf, VS Code, Raycast, and generic MCP configs.
- Redacted JSON scan and doctor output.
- Read-only backup dry-run plans.
- Read-only remediation plans for MCP/security findings.
- SARIF and policy output for CI.
- User-level nightly schedule generation.

## Next

- More MCP config shapes for Codex, Claude Code, and editor integrations.
- Richer TUI actions with real clipboard/export/open-doc behavior behind explicit keypresses.
- Golden SARIF snapshots and broader no-write tests.
- GoReleaser config after CI proves stable.
- Signed releases with Sigstore/Cosign.
- Screenshot and GIF assets for the README.

## Later

- Encrypted local snapshots.
- Cross-machine diff.
- Raycast extension.
- Private dotfiles integration.
- Docker/Unraid dashboard.
- GitHub Action wrapper for repository policy checks.
- Restore workflow only after snapshot, preview, rollback, and secret-safety controls exist.

## Non-Goals For V1

- Telemetry.
- Cloud dashboard.
- Live sync.
- Secret copying.
- Automatic config mutation.
- Git push automation.
