# Support Matrix

## Adapter Coverage

Nightward scans local HOME config and workspace config for these families:

| Family | Coverage |
| --- | --- |
| [Codex](https://github.com/openai/codex) | HOME and workspace config, including MCP server entries. |
| [Claude Code](https://docs.claude.com/en/docs/claude-code) and [Claude Desktop](https://claude.ai/download) | User and desktop MCP config shapes. |
| [Cursor](https://cursor.com/) | Global and project MCP JSON. |
| [Windsurf](https://windsurf.com/) | Cascade MCP config. |
| [VS Code](https://code.visualstudio.com/) | Workspace MCP JSON shapes. |
| [Raycast](https://www.raycast.com/) | Nightward extension preferences and local command posture. |
| [JetBrains](https://www.jetbrains.com/) | Known AI/MCP config paths when present. |
| [Zed](https://zed.dev/) | Known assistant/MCP config paths when present. |
| [Continue](https://www.continue.dev/) | Agent config paths and MCP-style entries. |
| [Cline](https://cline.bot/) / [Roo Code](https://roocode.com/) | MCP server JSON shapes. |
| [Aider](https://aider.chat/) | Known config/state paths. |
| [OpenCode](https://opencode.ai/) | Known config/state paths. |
| [Goose](https://block.github.io/goose/) | Known config/state paths. |
| [LM Studio](https://lmstudio.ai/) | Known local model/app state paths. |
| [Ollama](https://ollama.com/) / [Open WebUI](https://openwebui.com/) | Local model/auth/runtime paths. |
| [Neovim](https://neovim.io/) | Known AI plugin config paths. |
| Generic MCP files | Common `mcp.json`, `mcp_config.json`, `.mcp.json`, and workspace MCP variants. |

Run `nw adapters list --json` or `nw adapters explain <name>` for the exact paths checked on your machine.

## Provider Coverage

| Provider | Status | Install docs | Boundary |
| --- | --- | --- | --- |
| [Gitleaks](https://github.com/gitleaks/gitleaks) | Explicit local provider | [Install](https://github.com/gitleaks/gitleaks#installing) | Runs only with `--with gitleaks`. |
| [TruffleHog](https://github.com/trufflesecurity/trufflehog) | Explicit local provider | [Install](https://github.com/trufflesecurity/trufflehog#installation) | Runs only with `--with trufflehog`; verification is disabled by default. |
| [Semgrep](https://semgrep.dev/) | Explicit local provider | [Install](https://semgrep.dev/docs/getting-started/) | Runs only with `--with semgrep` and repo-local config. |
| [Trivy](https://trivy.dev/) | Explicit online-capable provider | [Install](https://trivy.dev/latest/getting-started/installation/) | Requires `--with trivy --online`; vulnerability DB behavior can contact upstream services. |
| [OSV-Scanner](https://google.github.io/osv-scanner/) | Explicit online-capable provider | [Install](https://google.github.io/osv-scanner/installation/) | Requires `--with osv-scanner --online`. |
| [Socket](https://socket.dev/) | Explicit online-capable provider | [CLI docs](https://docs.socket.dev/docs/socket-cli) | Requires `--with socket --online`; creates a remote scan artifact and uploads dependency manifest metadata. |

## Integration Coverage

| Integration | Status | Boundary |
| --- | --- | --- |
| CLI/TUI | Shipped | Local read-only scan/report flows plus explicit output/export files |
| [Raycast](/integrations/raycast) | Shipped in-repo | Read-only commands, menu-bar status, clipboard/report-folder actions |
| [MCP server](/integrations/mcp-server) | Shipped | Stdio-only read-only tools/resources; no listener, mutation tools, or online providers |
| [GitHub Action](/integrations/github-action) | Shipped | CI policy/SARIF output against repository fixtures/workspaces |
| [Trunk](/integrations/trunk) | Shipped | Repo-owned plugin definition; users pin to a Nightward tag or SHA |

## Platform Coverage

| Surface | macOS | Linux | Windows |
| --- | --- | --- | --- |
| CLI/TUI | Supported arm64/amd64 | Supported arm64/amd64 | Supported amd64; ARM64 deferred until the OpenTUI sidecar has a Bun compile target |
| Schedule install | launchd | systemd user timers | dry-run/fallback only |
| Raycast | Supported | Not applicable | Not applicable |
| MCP server | Supported arm64/amd64 | Supported arm64/amd64 | Supported amd64; ARM64 deferred with the release archive |
| GitHub Action/Trunk | Supported in CI | Supported in CI | Best-effort in CI |
