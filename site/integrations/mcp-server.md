# MCP Server

Nightward ships a read-only stdio MCP server for AI clients that support local MCP tools.

```sh
nw mcp serve
```

The server exposes Nightward context to AI tools without opening a network listener, mutating config, enabling telemetry, or running online-capable providers by default.

## Tools

| Tool | Purpose |
| --- | --- |
| `nightward_scan` | Run a read-only HOME or workspace scan. |
| `nightward_doctor` | Return adapters, schedule status, provider posture, and local Nightward version. |
| `nightward_findings` | List findings with optional severity, tool, rule, and search filters. |
| `nightward_explain_finding` | Return one finding by full ID or unique prefix. |
| `nightward_fix_plan` | Generate plan-only remediation for all findings, one finding, or one rule. |
| `nightward_report_changes` | Compare the latest two saved report JSON files. |
| `nightward_policy_check` | Run the policy gate with optional offline analysis. |

## Resources

- `nightward://rules`
- `nightward://providers`
- `nightward://schedule`
- `nightward://latest-report`

## Client Config

Generic stdio MCP config:

```json
{
  "mcpServers": {
    "nightward": {
      "command": "nw",
      "args": ["mcp", "serve"]
    }
  }
}
```

Claude Code and other local AI clients can use the same command/args shape when they support stdio MCP servers. Use an absolute path for `command` if the client does not inherit your shell `PATH`.

## Security Model

- Read-only tools only.
- No default network calls.
- No online-capable provider execution through MCP v1.
- No live config mutation or schedule install/remove tools.
- Tool output is bounded and redacted before it reaches the MCP client.
- Tool failures are returned as MCP tool results with `isError`, so the client can show the error without treating it as a protocol failure.

Use explicit CLI commands such as `nw analyze --with trivy --online --json` outside MCP when you intentionally want online provider execution.
