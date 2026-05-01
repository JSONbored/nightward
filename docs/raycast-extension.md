# Raycast Extension

Nightward's Raycast extension is a local read-only companion for the CLI/TUI.

## Location

```sh
integrations/raycast
```

## Commands

- `Nightward Dashboard`: scan counts, schedule status, adapter summary, and top findings.
- `Nightward Findings`: searchable findings with a severity filter, detail pane, and copy/open-doc actions.
- `Nightward Analysis`: offline analysis signals with severity, confidence, evidence, and recommended action.
- `Nightward Provider Doctor`: optional provider availability and privacy posture.
- `Explain Nightward Finding`: detail view for a known finding ID.
- `Explain Nightward Signal`: analysis signal view for a known finding ID.
- `Export Nightward Fix Plan`: copies `nw fix export --all --format markdown`.
- `Export Nightward Analysis`: copies a redacted offline analysis Markdown report.
- `Open Nightward Reports`: opens `~/.local/state/nightward/reports` when it already exists.

## Preferences

- `Nightward Command`: command or path to execute. Defaults to `nw` and falls back to `nightward` when `nw` is missing.
- `Home Override`: optional directory passed as `NIGHTWARD_HOME` for fixture scans or alternate local homes.

## Security Boundaries

The extension uses `execFile`, not a shell, for local Nightward commands. It calls only:

- `scan --json`
- `doctor --json`
- `findings list --json`
- `findings explain <id> --json`
- `fix plan --all --json`
- `fix export --all --format markdown`
- `analyze --all --json`
- `analyze finding <id> --json`
- `providers doctor --json`

It does not call schedule install/remove, backup writes, snapshot writes, restore, Git, or any config mutation command.

## Validation

```sh
cd integrations/raycast
npm ci
npm test
npm run lint
npm run build
```

Manual smoke:

```sh
npm run dev
```

Manual smoke must use a fixture `Home Override`, not a real local home, before screenshots or store metadata are published. Cover at least:

- Dashboard loads scan counts, schedule status, adapters, and top findings.
- Findings search/filter/detail panes render redacted evidence and docs actions.
- Analysis renders offline signals and provider warnings.
- Provider Doctor shows local provider status and does not run online-capable providers without explicit opt-in.
- Export commands copy redacted Markdown and do not mutate local config.
- Open Reports opens only an existing reports folder.

Record the fixture path, commit SHA, command result, and reviewer in `docs/screenshots.md` or adjacent release notes when screenshots/GIFs are captured.

Do not run `npm run publish` unless publishing is explicitly in scope.
