# Privacy Model

Nightward is local-first by design.

## Defaults

- No telemetry.
- No default network calls.
- No cloud dashboard.
- No live config mutation.
- No secret copying.
- Redacted reports by default.

## Writes

Read-only commands stay read-only unless an explicit output path is provided.

Expected write paths:

- `--output` or `--output-dir` reports.
- SARIF output files.
- Explicit redacted exports from the TUI/Raycast flows.
- User-level schedule install/remove commands.

Scheduled scans write redacted reports under `~/.local/state/nightward/reports/` and never push to Git or copy secrets.

## Optional providers

Provider execution is opt-in. Online-capable providers require explicit `--online` or policy opt-in. Nightward should explain provider privacy impact before using tools that may contact external services.
