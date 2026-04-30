# Privacy Model

Nightward is designed around local custody. The scanner inspects local file metadata and selected config contents, then emits redacted reports for the user to review.

## Runtime Boundaries

- No telemetry.
- No analytics.
- No cloud dashboard.
- No network calls from Nightward runtime.
- No live backup, restore, Git push, or secret copy.
- No agent config mutation in scan, doctor, findings, fix, policy, or backup-plan commands.
- The TUI can copy text, export redacted fix-plan Markdown, and open docs only after explicit keypresses.
- The Raycast extension calls only read-only Nightward commands and explicit clipboard/report-folder actions.

## Write Paths

Nightward writes only when explicitly asked:

- `scan --output FILE`
- `scan --output-dir DIR`
- `policy sarif --output FILE`
- TUI `e` key: redacted fix-plan export under `~/.local/state/nightward/exports`
- `schedule install`
- `schedule remove`

Schedule install/remove writes only user-level scheduler files. It does not copy configs, secrets, dotfiles, or reports into Git.

The TUI docs action opens an http(s) documentation URL through the OS default opener after the user presses `o`; Nightward itself does not fetch docs content.

The Raycast extension does not add a Nightward config write path. `Export Nightward Fix Plan` copies redacted Markdown to the clipboard after the user invokes that command. `Open Nightward Reports` opens the existing reports folder and does not create it.

## Redaction Rules

Nightward must not emit secret values in:

- scan JSON
- findings output
- fix-plan JSON
- Markdown fix exports
- SARIF output
- TUI detail views
- TUI fix-plan exports

Secret env handling distinguishes:

- env key references, such as `${API_TOKEN}`, which become guidance-only remediation
- inline secret values, which become higher-risk externalization plans

Secret header handling follows the same rule: header names may be emitted, but header values are never emitted. Inline values become higher-risk `mcp_secret_header` findings; environment references become guidance-only remediation.

MCP argument evidence redacts secret-looking assignments and flag values, such as `--api-key value`, `TOKEN=value`, and `Authorization: value`.

Remote MCP URL evidence is structural only. Nightward records scheme and host for review, strips path/query details, and does not call the endpoint.

## What Still Needs Human Review

Nightward can detect obvious local risk, but it cannot know user intent. Findings such as broad filesystem mounts, local token paths, shell wrappers, local MCP endpoints, and unknown MCP server shapes should be reviewed before syncing.

If a report contains private state, treat that as a bug and follow [SECURITY.md](../SECURITY.md).
