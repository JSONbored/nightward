# What Is Nightward?

Nightward is a local-first security and portability tool for AI agent state.

It scans common AI/devtool config locations, classifies what can safely move into a private dotfiles repo, highlights MCP trust-boundary issues, and produces redacted reports for humans, CI, Trunk, Raycast, and GitHub code scanning.

## What it checks

- AI agent and editor config paths.
- MCP server definitions.
- Local endpoint and filesystem assumptions.
- Sensitive env/header references.
- App-owned state, runtime caches, and credential material.
- Workspace AI config drift in repositories.

## What it does not do

- No telemetry.
- No default network calls.
- No secret copying.
- No Git push automation.
- No live agent-config mutation.
- No restore workflow until preview, rollback, and secret-safety controls are strong enough.

## Output surfaces

- TUI dashboard and detail panes.
- Redacted JSON for automation.
- SARIF for GitHub code scanning.
- Policy checks for CI.
- Trunk plugin rules.
- Raycast read-only companion commands.
- Dry-run backup and snapshot plans.
