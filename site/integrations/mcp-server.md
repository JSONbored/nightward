# MCP Server

<!-- markdownlint-disable MD033 -->

Nightward ships a stdio [Model Context Protocol](https://modelcontextprotocol.io/) server so AI clients can scan, explain, compare, plan, and apply bounded Nightward actions without sending users back to raw CLI commands.

```sh
nw mcp serve
```

The MCP surface is action-capable, but it is not an arbitrary file editor. Applies go only through the shared Nightward action registry and require the responsibility disclosure, `confirm: true`, redacted output, and audit logging.

## Client Setup

Install Nightward first, then add the same stdio command to your client. Use an absolute path such as `/Users/alice/.local/bin/nw` if the client does not inherit your shell `PATH`.

<McpClientTabs />

Client references: [Claude MCP](https://docs.claude.com/en/docs/claude-code/mcp), [Cursor MCP](https://docs.cursor.com/advanced/model-context-protocol), [Codex MCP config](https://developers.openai.com/learn/docs-mcp), [Windsurf Cascade MCP](https://docs.windsurf.com/windsurf/cascade/mcp), and [Google Antigravity](https://antigravity.google/)'s in-app "Manage MCP Servers -> View raw config" flow.

VS Code-style clients use a different key:

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

Restart or reload the AI client after editing its MCP config. A useful first prompt is: "Audit my AI setup with Nightward, explain the top risks, and do not apply actions unless I explicitly confirm them."

## Tools

| Tool | Purpose | Apply behavior |
| --- | --- | --- |
| `nightward_scan` | Run a redacted HOME or workspace scan. | Read-only. |
| `nightward_doctor` | Return provider, schedule, disclosure, and action posture. | Read-only. |
| `nightward_findings` | List findings with severity/rule filters. | Read-only. |
| `nightward_explain_finding` | Return one finding by ID or unique prefix. | Read-only. |
| `nightward_analysis` | Run Nightward analysis with selected providers. | Read-only; online providers require opt-in. |
| `nightward_explain_signal` | Return one analysis signal by ID or prefix. | Read-only. |
| `nightward_policy_check` | Run the policy gate with optional analysis. | Read-only. |
| `nightward_fix_plan` | Generate plan-only remediation directions. | Read-only. |
| `nightward_report_history` | List saved scheduled reports. | Read-only. |
| `nightward_report_changes` | Compare two report files or the latest two saved reports. | Read-only. |
| `nightward_actions_list` | List bounded registry actions. | Read-only. |
| `nightward_action_preview` | Preview one registry action. | Read-only. |
| `nightward_action_apply` | Apply one registry action, including policy init/ignore-with-reason and report/cache cleanup actions. | Requires disclosure acceptance and `confirm: true`. |
| `nightward_rules` | List rules and remediation metadata. | Read-only. |
| `nightward_providers` | List provider capabilities and status. | Read-only. |

For casual AI-client use, prefer compact calls:

```json
{
  "include_analysis": true,
  "compact": true,
  "limit": 25
}
```

Compact mode keeps pass/fail, threshold, summary counts, and bounded finding or signal metadata without flooding the model with every local detail.

## Resources

- `nightward://latest-summary`
- `nightward://latest-report`
- `nightward://rules`
- `nightward://providers`
- `nightward://schedule`
- `nightward://actions`
- `nightward://disclosure`
- `nightward://report-history`

## Prompts

- `audit_my_ai_setup`
- `explain_top_risks`
- `fix_this_finding`
- `set_up_providers`
- `compare_reports`

These prompts are workflow starters for clients that expose MCP prompts. They are deliberately cautious: they tell the assistant to preview registry actions first and avoid apply unless the user explicitly confirms.

## Safety Model

- Stdio only; no HTTP listener.
- No telemetry.
- No default network calls.
- Strict tool input schemas, server-side invalid-argument rejection, and structured output.
- Tool execution failures return `isError: true`, not protocol crashes.
- Online-capable providers stay blocked unless explicitly allowed.
- Direct apply is limited to shared action-registry IDs.
- No arbitrary MCP/agent config mutation in MCP v1.
- Explicit workspace and report-diff paths must stay under `NIGHTWARD_HOME`, exist as the expected regular file or directory type, and avoid symlink components.
- Apply output is redacted and every successful apply appends an audit event.

Use `nightward_action_preview` before `nightward_action_apply`. For package-manager provider installs, read the command, provider privacy boundary, and rollback expectations before confirming.

## Registry Package

Nightward's existing npm launcher is the MCP Registry package target:

- npm package: `@jsonbored/nightward`
- MCP registry name: `io.github.jsonbored/nightward`
- metadata: `server.json`
- package verification: `mcpName` in `packages/npm/package.json`

The registry publishes metadata, not binaries. Nightward still ships platform-specific CLI archives through the normal release path: macOS arm64/amd64, Linux arm64/amd64, and Windows amd64. Windows ARM64 remains deferred. Windows schedule install remains preview/fallback only.
