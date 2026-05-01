# Nightward Raycast Extension

Run Nightward scans, findings, and redacted fix-plan exports from Raycast.

This extension is read-only. It runs local `nw`/`nightward` commands, renders redacted JSON output, copies user-requested fix-plan text, and opens the scheduled report folder. It does not install schedules, edit agent configs, restore files, push to Git, or copy secret values.

## Commands

- `Nightward Dashboard`: scan summary, schedule status, adapters, and top findings.
- `Nightward Status`: menu-bar counter for findings, analysis signals, provider warnings, and scheduled-report state.
- `Nightward Findings`: searchable findings with severity filters and detail panes.
- `Nightward Analysis`: offline analysis signals and provider warnings.
- `Nightward Provider Doctor`: optional provider status without executing online providers.
- `Explain Nightward Finding`: detail view for a specific finding ID.
- `Explain Nightward Signal`: detail view for the analysis signal attached to a finding ID.
- `Export Nightward Fix Plan`: copies `nw fix export --format markdown` output.
- `Export Nightward Analysis`: copies redacted offline analysis markdown.
- `Open Nightward Reports`: opens `~/.local/state/nightward/reports` in Finder.

## Preferences

- `Nightward Command`: defaults to `nw` and falls back to `nightward` if the alias is unavailable.
- `Home Override`: optional `NIGHTWARD_HOME` override for testing alternate local homes.

## Development

```bash
npm ci
npm test
npm run lint
npm run build
```

Manual smoke testing requires the Raycast CLI:

```bash
npm run dev
```
