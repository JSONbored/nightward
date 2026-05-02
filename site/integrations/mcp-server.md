# MCP Server

<!-- markdownlint-disable MD033 -->

Nightward ships a read-only stdio [Model Context Protocol](https://modelcontextprotocol.io/) server so AI clients can ask for local scan, policy, rule, provider, report-history, and fix-plan context without receiving mutation tools.

```sh
nw mcp serve
```

Use the MCP server when you want an AI assistant to understand your local AI-tool risk posture. Keep normal CLI commands for release gates, CI, scheduled scans, online provider execution, and anything that should write files.

## Client Setup

Install Nightward first, then add the same stdio command to your client. Use an absolute path such as `/Users/alice/.local/bin/nw` if the client does not inherit your shell `PATH`.

<McpClientTabs />

Client references: [Claude MCP](https://docs.claude.com/en/docs/claude-code/mcp), [Cursor MCP](https://docs.cursor.com/advanced/model-context-protocol), [Codex MCP config](https://developers.openai.com/learn/docs-mcp), [Windsurf Cascade MCP](https://docs.windsurf.com/windsurf/cascade/mcp), and [Google Antigravity](https://antigravity.google/)’s in-app “Manage MCP Servers → View raw config” flow.

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

Restart or reload the AI client after editing its MCP config. Then ask for a safe first check such as “List my Nightward findings and summarize the critical ones.”

## Tools

| Tool | Purpose | Output shape |
| --- | --- | --- |
| `nightward_scan` | Run a read-only HOME or workspace scan. | Redacted scan report summary plus bounded findings. |
| `nightward_doctor` | Return adapters, schedule status, provider posture, and local Nightward version. | Doctor JSON. |
| `nightward_findings` | List findings with severity, tool, rule, and search filters. | Filtered finding list. |
| `nightward_explain_finding` | Return one finding by full ID or unique prefix. | One redacted finding with impact and recommendation. |
| `nightward_fix_plan` | Generate plan-only remediation for all findings, one finding, or one rule. | Redacted fix-plan JSON. |
| `nightward_report_changes` | Compare the latest two saved report JSON files. | Added, removed, and changed finding summary. |
| `nightward_policy_check` | Run the policy gate with optional offline analysis. | Policy result; pass `compact: true` for chat summaries. |

For casual AI-client use, prefer:

```json
{
  "strict": true,
  "include_analysis": true,
  "compact": true
}
```

Compact mode returns pass/fail, threshold, summary counts, and bounded violation metadata without flooding the model with every ignored item or full finding.

## Resources

- `nightward://rules`
- `nightward://providers`
- `nightward://schedule`
- `nightward://latest-report`

These resources are read-only. They are useful when an assistant needs rule/provider context before explaining a result.

## Safety Model

- Stdio only; no HTTP listener.
- Read-only tools only.
- No telemetry.
- No default network calls.
- No live config mutation or schedule install/remove tools.
- No online-capable provider execution through MCP v1.
- Tool output is bounded and redacted before it reaches the client.
- Tool failures return MCP tool results with `isError`, so client UI can show errors without treating them as protocol failures.

Use explicit CLI commands such as `nw analyze --with trivy --online --json` outside MCP when you intentionally want online provider execution.

## Publishing Status

Nightward’s MCP server is built into the CLI; there is no separate npm package today. The official discovery path is the [MCP Registry](https://registry.modelcontextprotocol.io/), which is currently preview infrastructure backed by the [modelcontextprotocol/registry](https://github.com/modelcontextprotocol/registry) project.

To publish Nightward there, the project needs a release artifact that can be referenced as an MCP server package, a `server.json` metadata file, namespace verification, and a `mcp-publisher publish` run. The registry hosts metadata, not binaries, so the practical next step is to decide whether Nightward’s npm launcher or a dedicated MCP-only package should be the registry package target.
