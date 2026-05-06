# Privacy Model

Nightward is designed around local custody. The scanner inspects local file metadata and selected config contents, then emits redacted reports for the user to review.

## Runtime Boundaries

- No telemetry.
- No Nightward runtime analytics.
- No cloud dashboard.
- No network calls from Nightward runtime.
- Offline analysis is the default. Local provider execution happens only when the user explicitly selects providers with `--with` or persisted provider settings. Online-capable providers stay blocked unless the user explicitly passes `--online` or opts in through policy/config/settings. `trivy`, `grype`, `osv-scanner`, `scorecard`, and `socket` are online-capable because vulnerability databases, repository checks, or remote scan artifacts can contact third-party services.
- No restore, Git push, sync, or secret copy.
- No agent config mutation in scan, doctor, findings, fix, policy, or backup-plan commands.
- The TUI can apply only shared action-registry operations after disclosure acceptance and an explicit confirmation keypress.
- The Raycast extension exposes the same shared action registry and uses Raycast confirmation prompts before applying actions.
- The MCP server is stdio-only and read-only. It exposes scan, analysis, policy, report, rule, provider, prompt, resource, action-list, and action-preview context, but cannot apply local writes.

## Write Paths

Nightward writes only when explicitly asked:

- `scan --output FILE`
- `policy sarif --output FILE`
- `policy sarif --output -` writes SARIF to stdout only.
- `disclosure accept` or first TUI acceptance writes `~/.config/nightward/settings.json`
- Confirmed provider settings and online-provider settings update `~/.config/nightward/settings.json`
- Confirmed provider installs run only through the shared action registry after preview, disclosure acceptance, and confirmation.
- Confirmed policy init/ignore actions write bounded Nightward policy files under `NIGHTWARD_HOME`; policy paths must be clean relative Nightward policy paths, and existing symlinks are rejected.
- Confirmed schedule install/remove writes or removes user-level launchd/systemd files only.
- Confirmed backup snapshots copy only regular portable backup candidates under `~/.local/state/nightward/snapshots`; symlinked or non-regular candidates are skipped and recorded in the manifest without following targets.
- Confirmed cleanup actions remove only Nightward-owned report, log, or cache entries.
- Raycast clipboard exports and report-folder open actions after explicit command invocation.

Confirmed action writes append audit events under `~/.local/state/nightward/audit.jsonl`. Nightward still does not restore config, push to Git, sync secrets, or rewrite live MCP/agent configs.

Nightward-owned state writes reject symlinked directories and symlinked files before writing settings, audit logs, schedules, snapshots, or action-managed policy files.

`nw mcp serve` cannot apply local writes. MCP clients cannot accept the Nightward disclosure, request arbitrary file edits, apply shared registry actions, live MCP/agent config rewrites, restore operations, Git pushes, or secret sync. MCP can list and preview shared action-registry operations so the user can apply them out-of-band in the CLI, TUI, or Raycast extension. MCP tool arguments are validated server-side against strict schemas, and MCP workspace/report paths must stay under `NIGHTWARD_HOME`, exist as regular files or directories as appropriate, and avoid symlink components.

The TUI docs action opens an http(s) documentation URL through the OS default opener after the user presses `o`; Nightward itself does not fetch docs content.

The Raycast extension's write path is limited to the shared Nightward action registry. `Export Nightward Fix Plan` copies redacted Markdown to the clipboard after the user invokes that command. `Open Nightward Reports` opens the existing reports folder and does not create it.

## Public Website Analytics

Nightward's runtime privacy model is separate from the public docs/marketing website. The CLI, TUI, MCP server, Raycast extension, npm launcher, and local docs preview do not send analytics.

The deployed website may use explicitly configured, self-hosted Umami for aggregate visitor analytics. That script is build-time gated and is absent unless the Pages build receives Umami environment values. When enabled, it is configured for `nightward.aethereal.dev`, respects browser Do Not Track, and excludes URL search parameters and hash fragments.

Do not add analytics keys, tracker URLs, or visitor identifiers to Nightward reports, runtime config, README media, fixture captures, or local development defaults.

## Redaction Rules

Nightward must not emit secret values in:

- scan JSON
- findings output
- fix-plan JSON
- analysis JSON
- Markdown fix exports
- Markdown analysis exports
- SARIF output
- TUI detail views
- TUI fix-plan exports

Secret env handling distinguishes:

- env key references, such as `${API_TOKEN}`, which become guidance-only remediation
- inline secret values, which become higher-risk externalization plans

Secret header handling follows the same rule: header names may be emitted, but header values are never emitted. Inline values become higher-risk `mcp_secret_header` findings; environment references become guidance-only remediation.

MCP argument evidence redacts secret-looking assignments and flag values, such as `--api-key value`, `TOKEN=value`, and `Authorization: value`.

Remote MCP URL evidence is structural only. Nightward records scheme and host for review, strips path/query details, and does not call the endpoint.

Provider doctor output is intentionally about availability and privacy posture. It does not run optional scanners by default, install missing tools, or send package/file metadata to online services. Explicit provider runs use timeouts, bounded output capture, and redacted finding metadata; online-capable providers require `--online` or policy/settings opt-in. Provider install actions run only after disclosure acceptance and confirmation.

## What Still Needs Human Review

Nightward can detect obvious local risk, but it cannot know user intent. Findings such as broad filesystem mounts, local token paths, shell wrappers, local MCP endpoints, and unknown MCP server shapes should be reviewed before syncing.

If a report contains private state, treat that as a bug and follow [SECURITY.md](../SECURITY.md).
