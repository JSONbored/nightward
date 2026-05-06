# Nightward MCP Server

Nightward includes a stdio MCP server:

```sh
nw mcp serve
```

The server is a first-class Nightward surface for AI clients. It exposes scan, analysis, policy, report, provider, rule, prompt, and bounded action workflows without requiring users to copy CLI commands out of chat.

## Protocol Behavior

- Negotiates MCP `2025-11-25` and remains compatible with `2025-06-18`.
- Declares `tools`, `resources`, and `prompts` capabilities.
- Provides strict tool input schemas with `additionalProperties: false`.
- Enforces those input schemas server-side, including unknown-key rejection, required fields, type checks, severity enums, `confirm: true`, and integer bounds.
- Returns `structuredContent` plus text fallback for tool results.
- Reports tool execution failures as MCP tool results with `isError: true`.
- Adds output schemas and tool annotations for read-only, destructive, idempotent, and open-world hints.

## Exposed Tools

- `nightward_scan`
- `nightward_doctor`
- `nightward_findings`
- `nightward_explain_finding`
- `nightward_analysis`
- `nightward_explain_signal`
- `nightward_policy_check`
- `nightward_fix_plan`
- `nightward_report_history`
- `nightward_report_changes`
- `nightward_actions_list`
- `nightward_action_preview`
- `nightward_action_apply`
- `nightward_rules`
- `nightward_providers`

`nightward_action_apply` is intentionally narrow. It can apply only shared Nightward action-registry IDs, such as disclosure acceptance, policy init/ignore-with-reason, schedule install/remove where supported, backup snapshot, report/cache cleanup, provider install/enable/disable, and online-provider toggles. It cannot rewrite arbitrary MCP or agent config.

Apply calls require:

- accepted Nightward responsibility disclosure
- `confirm: true`
- action availability checks
- redacted output
- audit logging under Nightward state

## Exposed Resources

- `nightward://latest-summary`
- `nightward://latest-report`
- `nightward://rules`
- `nightward://providers`
- `nightward://schedule`
- `nightward://actions`
- `nightward://disclosure`
- `nightward://report-history`

## Exposed Prompts

- `audit_my_ai_setup`
- `explain_top_risks`
- `fix_this_finding`
- `set_up_providers`
- `compare_reports`

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

VS Code-style clients use `servers` plus `type`:

```json
{
  "servers": {
    "nightward": {
      "type": "stdio",
      "command": "nw",
      "args": ["mcp", "serve"]
    }
  }
}
```

Use an absolute `command` path if the AI client does not inherit the same `PATH` as your login shell.

## Registry Metadata

Nightward uses the existing npm launcher as the MCP Registry package target:

- package: `@jsonbored/nightward`
- registry name: `io.github.jsonbored/nightward`
- package field: `mcpName`
- metadata file: `server.json`

CI validates that `server.json` and `packages/npm/package.json` agree before the npm package is considered release-ready.

## Safety Rules

- Stdio only; no HTTP listener.
- No telemetry.
- No default network calls.
- Online-capable providers remain blocked unless explicitly allowed.
- Direct apply is limited to the shared action registry.
- No live MCP/agent config mutation in MCP v1.
- Workspace and explicit report-diff paths must stay under `NIGHTWARD_HOME`, exist as the expected regular file or directory type, avoid symlink components, and pass the existing bounded report-size checks.
- Tool/resource/prompt output is bounded and redacted before it reaches the client.
