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

- Explicit `--output` report/SARIF files.
- Explicit redacted exports from the TUI/Raycast flows.
- Explicit clipboard/report-folder actions from Raycast.

Schedule install/remove is plan-only in v1. It describes intended launchd/systemd/cron commands and does not install timers, push to Git, or copy secrets.

## Optional providers

Provider execution is opt-in. Local providers (`gitleaks`, `trufflehog`, and repo-configured `semgrep`) require explicit `--with`. Online-capable providers (`trivy`, `osv-scanner`, and `socket`) require explicit `--with` plus `--online` or policy opt-in.

`socket` creates a remote Socket scan artifact from dependency manifest metadata. Nightward records only redacted returned metadata and does not fetch remote Socket reports in v1.

## Website analytics

Nightward runtime surfaces do not send analytics. The public docs website may load self-hosted Umami only when the production build is explicitly configured with Umami environment values.

When enabled, the website tracker is scoped to `nightward.aethereal.dev`, respects browser Do Not Track, and excludes URL search parameters and hash fragments. Local docs builds and previews stay analytics-free by default.
