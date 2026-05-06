# Privacy Model

Nightward is local-first by design.

## Defaults

- No telemetry.
- No default network calls.
- No cloud dashboard.
- No live config mutation without explicit action confirmation.
- No secret copying.
- Redacted reports by default.

## Writes

Read-only commands stay read-only unless an explicit output path is provided or the user applies a confirmation-gated action.

Expected write paths:

- Explicit `--output` report/SARIF files.
- Explicit redacted exports from the TUI/Raycast flows.
- Explicit clipboard/report-folder actions from Raycast.
- Confirmed action writes such as action-registry provider installs, provider settings, scheduled scan install/remove, bounded policy updates, and local backup snapshots.

Schedule install/remove uses user-level launchd or systemd user jobs where supported. Cleanup actions remove only Nightward-owned report, log, or cache entries. Action-managed writes reject symlinked Nightward-owned state paths, and backup snapshots skip symlinked or non-regular candidates without following their targets. These actions do not install root daemons, push to Git, or copy secrets.

The MCP server validates tool inputs on the server side and scopes explicit workspace/report paths under `NIGHTWARD_HOME` with regular-file/directory and no-symlink checks.

## Responsibility disclosure

Nightward is beta local operator tooling. Before TUI write actions run, users must accept that they are responsible for reviewing previews, confirmations, backups, provider behavior, and any resulting system changes. Nightward provides no warranty, and maintainers are not liable for broken configs, lost data, exposed secrets, package-manager side effects, or third-party tool behavior.

## Optional providers

Provider execution is opt-in. Local providers (`gitleaks`, `trufflehog`, repo-configured `semgrep`, and `syft`) require explicit `--with`. Online-capable providers (`trivy`, `osv-scanner`, `grype`, `scorecard`, and `socket`) require explicit `--with` plus `--online` or policy/settings opt-in.

`socket` creates a remote Socket scan artifact from dependency manifest metadata. Nightward records only redacted returned metadata and does not fetch remote Socket reports in v1.

## Website analytics

Nightward runtime surfaces do not send analytics. The public docs website may load self-hosted Umami only when the production build is explicitly configured with Umami environment values.

When enabled, the website tracker is scoped to `nightward.aethereal.dev`, respects browser Do Not Track, and excludes URL search parameters and hash fragments. Local docs builds and previews stay analytics-free by default.
