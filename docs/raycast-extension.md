# Raycast Extension

Nightward's Raycast extension is a local companion for the CLI/TUI. Most commands are review-only; `Nightward Actions` can apply confirmation-gated local writes through the shared CLI action registry.

## Location

```sh
integrations/raycast
```

## Commands

- `Nightward Dashboard`: scan counts, schedule status, adapter summary, and top findings.
- `Nightward Status`: compact menu-bar finding count, plus full critical/high/total counts, analysis signals, provider warnings, scheduled-report state, and latest-report access in the dropdown.
- `Nightward Findings`: searchable findings with a severity filter, detail pane, scoped fix-plan exports, reviewed-policy-ignore snippets, redacted evidence copy, and open-doc actions.
- `Nightward Analysis`: built-in offline signals plus explicitly selected providers.
- `Nightward Provider Doctor`: optional provider availability, privacy posture, install guidance for missing tools, and Raycast Analysis enable/disable controls.
- `Nightward Actions`: preview and apply confirmed provider, policy, schedule, backup, cleanup, and setup actions.
- `Nightward MCP Approvals`: approve or deny exact Model Context Protocol (MCP)-requested action tickets.
- `Explain Nightward Finding`: detail view for a known finding ID.
- `Explain Nightward Signal`: analysis signal view for a known finding ID.
- `Export Nightward Fix Plan`: copies `nw fix export --all --format markdown`.
- `Export Nightward Analysis`: copies a redacted offline analysis Markdown report.
- `Open Nightward Reports`: opens `~/.local/state/nightward/reports` when it already exists. Dashboard and menu-bar actions can reveal/open the latest report when `nw doctor --json` reports one.

## Preferences

- `Nightward Command`: command or path to execute. Defaults to `nw` and falls back to `nightward` when `nw` is missing.
- `Home Override`: optional typed path passed as `NIGHTWARD_HOME` for fixture scans or alternate local homes.
- `Allow Online Providers`: optional gate for selected online-capable providers. Socket creates a remote scan artifact when it runs.

## Security Boundaries

The extension uses `execFile`, not a shell, for local Nightward commands. It calls only:

- `scan --json`
- `doctor --json`
- `findings list --json`
- `findings explain <id> --json`
- `fix plan --json`
- `fix export --all --format markdown`
- `fix export --finding <id> --format markdown`
- `fix export --rule <rule> --format markdown`
- `analyze [--with providers] [--online] --json`
- `analyze finding <id> --json`
- `providers doctor [--with providers] [--online] --json`

Write-capable calls are limited to `actions apply <id> --confirm` through the shared action registry and `approvals approve|deny <approval-id>` through the Nightward approval queue, the human-reviewed queue for approve/deny decisions on MCP-requested actions. Provider Doctor previews `provider.install.<name>` and applies that registry action only after explicit confirmation; it no longer runs package-manager commands through a shell. MCP Approvals can approve a one-time ticket for MCP to run the specific approved registry action later, but it cannot approve disclosure, exfiltrate secrets, expose environment variables, read private keys, authorize arbitrary edits, perform bulk code changes, run raw shell commands, restore files, push Git state, or mutate live agent config outside the approved registry action.

Nightward is beta operator tooling. Users are responsible for reviewing confirmations, write targets, provider behavior, and package-manager side effects before applying actions. The project provides no warranty.

The menu-bar command runs the same read-only scan, doctor, and built-in offline analysis commands. Provider selections affect the Analysis and Export Analysis commands only. Online-capable selections remain blocked unless the user enables `Allow Online Providers`; the extension never enables background mutation.

## Validation

```sh
cd integrations/raycast
npm ci
npm test
npm run lint
npm run build
npm run store-check
```

Fixture UI validation:

```sh
npm run dev
```

Manual UI validation must use a fixture `Home Override`, not a real local home, before screenshots or store metadata are published. Cover at least:

- Dashboard loads scan counts, schedule status, adapters, and top findings.
- Menu-bar status shows finding, analysis, provider-warning, and schedule counters; its actions open existing read-only commands, open the latest report when present, and copy a redacted summary.
- Findings search/filter/detail panes render redacted evidence, docs actions, scoped fix-plan exports, and reviewed-policy-ignore snippets.
- Analysis renders built-in signals, selected provider output, provider warnings, and blocked-online-provider state.
- Provider Doctor shows provider status, install guidance, action-registry provider CLI installation, and enable/disable controls for Raycast Analysis without running online-capable providers unless explicit opt-in is enabled.
- Nightward Actions lists action IDs, risk, writes, commands, blocked reasons, and applies only after confirmation.
- Nightward MCP Approvals lists approval IDs, requested actions, status, expiry, writes, and commands; approve/deny requires a Raycast confirmation prompt.
- Export commands copy redacted Markdown and do not mutate local config.
- Open Reports opens only an existing reports folder.

Record the fixture path, commit SHA, command result, and reviewer in `docs/screenshots.md` or adjacent release notes when screenshots/GIFs are captured.

Do not run `npm run publish` unless publishing is explicitly in scope.

## Store Submission Readiness

`npm run store-check` verifies the self-contained package shape, required manifest fields, matching command source files, 512x512 manifest icon, README, CHANGELOG, and metadata screenshot count. It reports screenshot gaps as warnings so regular local validation can pass before manual Raycast capture is complete.

Use the strict gate immediately before preparing a draft PR to `raycast/extensions`:

```sh
npm run store-check:strict
```

Current expected blocker: the package still needs to be copied into a synced `raycast/extensions` fork and reviewed as a store PR. Fixture metadata screenshots are present under `integrations/raycast/metadata/`; re-capture them from `ray develop` with fixture `Home Override` whenever the UI changes.

Draft submission prep:

```sh
# In a local fork of raycast/extensions after syncing upstream:
mkdir -p extensions/nightward
rsync -a --delete \
  --exclude node_modules \
  --exclude dist \
  /path/to/nightward/integrations/raycast/ \
  extensions/nightward/
cd extensions/nightward
npm ci
npm test
npm run lint
npm run build
npm run store-check:strict
```
