# Audit An MCP-Heavy Workstation

Nightward treats MCP config as a trust boundary, not just editor settings.

## Run

```sh
nw scan --json
nw findings list
nw findings explain mcp_unpinned_package
nw analyze --json
nw providers doctor --with gitleaks,trufflehog,semgrep --json
```

## What To Look For

| Finding | Why it matters |
| --- | --- |
| `mcp_unpinned_package` | Package executors can drift between machines or over time. |
| `mcp_shell_command` | Shell wrappers hide what actually runs. |
| `mcp_secret_env` and `mcp_secret_header` | Inline credentials can leak through dotfiles, screenshots, logs, and support requests. |
| `mcp_broad_filesystem` | Broad roots give tools more local file reach than many users intend. |
| `mcp_local_endpoint` | Local services often assume machine-local trust. |

## Optional Providers

Local providers stay opt-in:

```sh
nw analyze --workspace . --with gitleaks,trufflehog,semgrep --json
```

Online-capable providers require an explicit online gate:

```sh
nw analyze --workspace . --with trivy,osv-scanner,socket --online --json
```
