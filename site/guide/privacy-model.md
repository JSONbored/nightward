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

Provider execution is opt-in. Local providers (`gitleaks`, `trufflehog`, and repo-configured `semgrep`) require explicit `--with`. Online-capable providers (`trivy`, `osv-scanner`, and `socket`) require explicit `--with` plus `--online` or policy opt-in.

`socket` creates a remote Socket scan artifact from dependency manifest metadata. Nightward records only redacted returned metadata and does not fetch remote Socket reports in v1.
