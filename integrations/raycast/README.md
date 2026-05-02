# Nightward Raycast Extension

Run Nightward scans, findings, provider checks, and redacted fix-plan exports from Raycast.

This extension is read-only. It runs local `nw`/`nightward` commands, renders redacted JSON output, copies user-requested fix-plan and policy-review text, and opens the scheduled report folder. It does not install schedules, edit agent configs, restore files, push to Git, or copy secret values.

## Commands

- `Nightward Dashboard`: scan summary, schedule status, adapters, and top findings.
- `Nightward Status`: compact menu-bar marker such as `3C`, `18H`, or a total count, with detailed findings, analysis signals, provider warnings, and scheduled-report state in the dropdown.
- `Nightward Findings`: searchable findings with severity filters, detail panes, scoped fix-plan exports, reviewed-policy-ignore snippets, and redacted evidence copy.
- `Nightward Analysis`: built-in offline analysis plus any providers explicitly selected in Provider Doctor.
- `Nightward Provider Doctor`: provider availability, privacy posture, and Raycast Analysis enable/disable controls.
- `Explain Nightward Finding`: detail view for a specific finding ID.
- `Explain Nightward Signal`: detail view for the analysis signal attached to a finding ID.
- `Export Nightward Fix Plan`: copies `nw fix export --format markdown` output.
- `Export Nightward Analysis`: copies redacted offline analysis markdown.
- `Open Nightward Reports`: opens `~/.local/state/nightward/reports` in Finder.

## Preferences

- `Nightward Command`: defaults to `nw` and falls back to `nightward` if the alias is unavailable.
- `Home Override`: optional typed `NIGHTWARD_HOME` path for testing alternate local homes.
- `Allow Online Providers`: opt-in gate for selected online-capable providers. Socket creates a remote scan artifact.

## Development

```bash
npm ci
npm test
npm run lint
npm run build
npm run store-check
```

Manual smoke testing requires the Raycast CLI:

```bash
npm run dev
```

Store submission should use `npm run store-check:strict` only after fixture-only metadata screenshots have been captured into `metadata/`.
