# Nightward MCP Server

Nightward includes a read-only stdio MCP server:

```sh
nw mcp serve
```

The server follows the MCP JSON-RPC lifecycle for `initialize`, `tools/list`, `tools/call`, `resources/list`, and `resources/read`. It is intentionally narrow for v1: no HTTP listener, no telemetry, no live config mutation, no schedule install/remove, and no online-capable provider execution.

## Exposed Tools

- `nightward_scan`
- `nightward_doctor`
- `nightward_findings`
- `nightward_explain_finding`
- `nightward_fix_plan`
- `nightward_report_changes`
- `nightward_policy_check`

## Exposed Resources

- `nightward://rules`
- `nightward://providers`
- `nightward://schedule`
- `nightward://latest-report`

## Example Client Config

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

Use an absolute `command` path if the AI client does not inherit the same `PATH` as your login shell.

## Safety Rules

- Output is bounded before it is returned to the MCP client.
- Nightward reports provider or tool execution errors as tool results with `isError: true`.
- The MCP server can include offline analysis, but it does not enable online providers in v1.
- Any future write-capable MCP tool must require a separate design review, confirmation UX, rollback story, and redaction tests.
