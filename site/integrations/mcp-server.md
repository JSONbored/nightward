# MCP Server

<!-- markdownlint-disable MD033 -->

Nightward ships a stdio [Model Context Protocol](https://modelcontextprotocol.io/) server so AI clients can scan, explain, compare, plan, preview Nightward's bounded actions, and request local approval for exact write tickets.

```sh
nw mcp serve
```

Most MCP tools are read-only. For writes, MCP can request a bounded action approval; the user approves or denies it locally in the Nightward CLI, TUI, or Raycast extension; then MCP can apply only that exact approved one-time ticket. MCP cannot accept the beta responsibility disclosure. That disclosure is Nightward's local one-time acknowledgement that write-capable beta actions are user-authorized, and MCP cannot self-confirm writes.

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

Restart or reload the AI client after editing its MCP config. A useful first prompt is: "Audit my AI setup with Nightward, explain the top risks, preview relevant actions, and request approval before applying any write action."

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
| `nightward_action_request` | Request local approval for one exact registry action. | Writes Nightward approval state only. |
| `nightward_action_status` | Read one approval ticket status. | Read-only. |
| `nightward_action_apply_approved` | Apply one approved, unexpired, one-time ticket. | Destructive only after local approval. |
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
- `nightward://action-approvals`
- `nightward://report-history`

## Prompts

- `audit_my_ai_setup`
- `explain_top_risks`
- `fix_this_finding`
- `set_up_providers`
- `compare_reports`

These prompts are workflow starters for clients that expose MCP prompts. They tell the assistant to preview registry actions, request approval when a bounded action is useful, and only apply exact tickets already approved locally.

## Safety Model

- Stdio only; no HTTP listener.
- No telemetry.
- No default network calls.
- Strict tool input schemas, server-side invalid-argument rejection, and structured output.
- Tool execution failures return `isError: true`, not protocol crashes.
- Online-capable providers stay blocked unless explicitly allowed.
- MCP cannot self-confirm local writes.
- MCP cannot accept the beta responsibility disclosure, Nightward's local one-time acknowledgement that write-capable beta actions are user-authorized.
- MCP approval requests record only Nightward-owned approval state. Applying an approved action consumes one exact one-time ticket and is audited.
- No arbitrary MCP/agent config mutation in MCP v1.
- Explicit workspace and report-diff paths must stay under `NIGHTWARD_HOME`, exist as the expected regular file or directory type, and avoid symlink components.
- Preview output is redacted and exposes write targets before any out-of-band apply.

Use `nightward_action_preview` before `nightward_action_request`. For provider installs that run a package manager such as Homebrew, Go, or npm, review the exact command, provider privacy boundary, and rollback expectations before approving the pending ticket in the CLI, TUI, or Raycast extension.

## Registry Package

Nightward's existing npm launcher is the MCP Registry package target:

- npm package: `@jsonbored/nightward`
- MCP registry name: `io.github.jsonbored/nightward`
- metadata: `server.json`
- package verification: `mcpName` in `packages/npm/package.json`

The registry publishes metadata, not binaries. Nightward still ships platform-specific CLI archives through the normal release path: macOS arm64/amd64, Linux arm64/amd64, and Windows amd64. Windows ARM64 remains deferred. Windows schedule install remains preview/fallback only.
