# Privacy Model

Nightward is designed around local custody. The scanner inspects local file metadata and selected config contents, then emits redacted reports for the user to review.

## Runtime Boundaries

- No telemetry.
- No analytics.
- No cloud dashboard.
- No network calls from Nightward runtime.
- No live backup, restore, Git push, or secret copy.
- No agent config mutation in scan, doctor, findings, fix, policy, or backup-plan commands.
- The Raycast extension calls only read-only Nightward commands and explicit clipboard/report-folder actions.

## Write Paths

Nightward writes only when explicitly asked:

- `scan --output FILE`
- `scan --output-dir DIR`
- `policy sarif --output FILE`
- `schedule install`
- `schedule remove`

Schedule install/remove writes only user-level scheduler files. It does not copy configs, secrets, dotfiles, or reports into Git.

The Raycast extension does not add a Nightward config write path. `Export Nightward Fix Plan` copies redacted Markdown to the clipboard after the user invokes that command. `Open Nightward Reports` opens the existing reports folder and does not create it.

## Redaction Rules

Nightward must not emit secret values in:

- scan JSON
- findings output
- fix-plan JSON
- Markdown fix exports
- SARIF output
- TUI detail views

Secret env handling distinguishes:

- env key references, such as `${API_TOKEN}`, which become guidance-only remediation
- inline secret values, which become higher-risk externalization plans

MCP argument evidence redacts secret-looking assignments and flag values, such as `--api-key value`, `TOKEN=value`, and `Authorization: value`.

## What Still Needs Human Review

Nightward can detect obvious local risk, but it cannot know user intent. Findings such as broad filesystem mounts, local token paths, shell wrappers, and unknown MCP server shapes should be reviewed before syncing.

If a report contains private state, treat that as a bug and follow [SECURITY.md](../SECURITY.md).
