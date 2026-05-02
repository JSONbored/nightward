# Raycast Extension

Nightward's Raycast extension is a local read-only companion for the CLI/TUI.

## Location

```sh
integrations/raycast
```

## Commands

- `Nightward Dashboard`: scan counts, schedule status, adapter summary, and top findings.
- `Nightward Status`: compact menu-bar severity marker such as `3C`, `18H`, or `59`, with full critical/high/total counts, analysis signals, provider warnings, scheduled-report state, and latest-report access in the dropdown.
- `Nightward Findings`: searchable findings with a severity filter, detail pane, scoped fix-plan exports, reviewed-policy-ignore snippets, redacted evidence copy, and open-doc actions.
- `Nightward Analysis`: built-in offline signals plus explicitly selected providers.
- `Nightward Provider Doctor`: optional provider availability, privacy posture, and Raycast Analysis enable/disable controls.
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

It does not call schedule install/remove, backup writes, snapshot writes, restore, Git, or any config mutation command.

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

Manual smoke:

```sh
npm run dev
```

Manual smoke must use a fixture `Home Override`, not a real local home, before screenshots or store metadata are published. Cover at least:

- Dashboard loads scan counts, schedule status, adapters, and top findings.
- Menu-bar status shows finding, analysis, provider-warning, and schedule counters; its actions open existing read-only commands, open the latest report when present, and copy a redacted summary.
- Findings search/filter/detail panes render redacted evidence, docs actions, scoped fix-plan exports, and reviewed-policy-ignore snippets.
- Analysis renders built-in signals, selected provider output, provider warnings, and blocked-online-provider state.
- Provider Doctor shows provider status and lets users enable or disable providers for Raycast Analysis without running online-capable providers unless explicit opt-in is enabled.
- Export commands copy redacted Markdown and do not mutate local config.
- Open Reports opens only an existing reports folder.

Record the fixture path, commit SHA, command result, and reviewer in `docs/screenshots.md` or adjacent release notes when screenshots/GIFs are captured.

Do not run `npm run publish` unless publishing is explicitly in scope.

## Store Submission Readiness

`npm run store-check` verifies the self-contained package shape, required manifest fields, matching command source files, 512x512 manifest icon, README, CHANGELOG, and metadata screenshot count. It reports missing screenshots as warnings so regular local validation can pass before manual Raycast capture is complete.

Use the strict gate immediately before preparing a draft PR to `raycast/extensions`:

```sh
npm run store-check:strict
```

Current expected blocker: store metadata screenshots are not committed yet. Capture them from `ray develop` with fixture `Home Override` and Raycast Window Capture before opening the draft submission PR.

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
