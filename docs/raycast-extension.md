# Raycast Extension

Nightward's Raycast extension is a local read-only companion for the CLI/TUI.

## Location

```sh
integrations/raycast
```

## Commands

- `Nightward Dashboard`: scan counts, schedule status, adapter summary, and top findings.
- `Nightward Findings`: searchable findings with a severity filter, detail pane, and copy/open-doc actions.
- `Explain Nightward Finding`: detail view for a known finding ID.
- `Export Nightward Fix Plan`: copies `nw fix export --all --format markdown`.
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

Do not run `npm run publish` unless publishing is explicitly in scope.
