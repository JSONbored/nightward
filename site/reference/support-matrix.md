# Support Matrix

## Adapter Coverage

Nightward scans local HOME config and workspace config for these families:

- Codex
- Claude Code and Claude Desktop
- Cursor
- Windsurf
- VS Code
- Raycast
- JetBrains
- Zed
- Continue
- Cline/Roo
- Aider
- OpenCode
- Goose
- LM Studio
- Ollama/Open WebUI
- Neovim
- Generic MCP files

Run `nw adapters list --json` or `nw adapters explain <name>` for the exact paths checked on your machine.

## Provider Coverage

- Explicit local providers: `gitleaks`, `trufflehog`, `semgrep`.
- Explicit online-capable providers: `trivy`, `osv-scanner`, `socket`.
- `socket` is remote scan creation, not a local-only provider.

## Integration Coverage

| Integration | Status | Boundary |
| --- | --- | --- |
| CLI/TUI | Shipped | Local read-only scan/report flows plus explicit output/export files |
| Raycast | Shipped in-repo | Read-only commands, menu-bar status, clipboard/report-folder actions |
| MCP server | Shipped | Stdio-only read-only tools/resources; no listener, mutation tools, or online providers |
| GitHub Action/Trunk | Shipped | CI policy/SARIF output against repository fixtures/workspaces |

## Platform Coverage

| Surface | macOS | Linux | Windows |
| --- | --- | --- | --- |
| CLI/TUI | Supported | Supported | Best-effort from Go builds |
| Schedule install | launchd | systemd user timers | dry-run/fallback only |
| Raycast | Supported | Not applicable | Not applicable |
| MCP server | Supported | Supported | Best-effort from Go builds |
| GitHub Action/Trunk | Supported in CI | Supported in CI | Best-effort in CI |
