# Analysis

Nightward analysis turns scan findings and classified paths into explainable risk signals.

Default behavior is offline:

```sh
nw analyze --json
nw analyze --workspace . --json
```

Analysis output includes:

- `subjects`: findings, inventory items, or packages under review
- `signals`: severity, category, confidence, evidence, and recommended action
- `providers`: built-in and optional provider status
- `summary`: counts by severity, category, and provider

Nightward avoids "safe" claims. A clean analysis means Nightward found no known risk signals from the enabled providers, not that the subject is trustworthy.

## Providers

The built-in `nightward` provider is always offline and enabled by default. Optional providers are discovered but not installed or run automatically.

| Provider | Execution class | Required gate | Scope |
| --- | --- | --- | --- |
| `gitleaks` | local command | `--with gitleaks` | Secret pattern scanning over the selected workspace. |
| `trufflehog` | local command | `--with trufflehog` | Filesystem secret scanning with verification disabled by default. |
| `semgrep` | local command | `--with semgrep` | Static analysis using only a repo-local Semgrep config. |
| `syft` | local command | `--with syft` | SBOM and local package inventory. |
| `trivy` | online-capable command | `--with trivy --online` | Filesystem vulnerability, secret, and misconfiguration scan; Trivy may update vulnerability databases. |
| `osv-scanner` | online-capable command | `--with osv-scanner --online` | Recursive source/lockfile vulnerability scan against OSV data. |
| `grype` | online-capable command | `--with grype --online` | Filesystem/SBOM vulnerability scanning; vulnerability DB behavior can contact upstream services. |
| `scorecard` | online-capable command | `--with scorecard --online` | OpenSSF repository trust checks against a git remote or `NIGHTWARD_SCORECARD_REPO`. |
| `socket` | online-capable command | `--with socket --online` | Creates a remote Socket scan artifact from dependency manifest metadata. |

Check provider posture:

```sh
nw providers list --json
nw providers doctor --json
nw providers doctor --with syft,socket --json
```

Unselected optional providers report `skipped`; selected online-capable providers report `blocked` until the online gate is present.

Online-capable providers remain blocked unless explicitly allowed:

```sh
nw providers doctor --with trivy,grype,scorecard,socket --online --json
nw analyze --workspace . --with trivy,osv-scanner,grype,scorecard,socket --online --json
```

Supported local providers can be executed explicitly during analysis:

```sh
nw analyze --workspace . --with gitleaks --json
nw analyze --workspace . --with gitleaks,trufflehog,semgrep,syft --json
```

Provider runs use timeouts and bounded output capture. Oversized stdout fails closed as a provider warning instead of being partially parsed. Nightward records redacted finding metadata, not raw secret values. Online-capable providers such as `trivy`, `osv-scanner`, `grype`, `scorecard`, and `socket` stay blocked unless the user also opts into online-capable behavior. Socket support is deliberately limited to scan creation and returned JSON parsing in v1; Nightward does not fetch or normalize remote Socket reports after creating the scan.

`semgrep` execution is local-config only. Nightward looks for `semgrep.yml`, `semgrep.yaml`, `.semgrep.yml`, `.semgrep.yaml`, or `.semgrep/config.yml` in the scanned workspace instead of using automatic rule discovery.

Policy/SARIF config can opt into provider execution:

```yaml
include_analysis: true
analysis_threshold: high
analysis_providers:
  - gitleaks
allow_online_providers: false
```

## Policy And SARIF

Analysis signals are advisory unless included explicitly. When included, signals at or above `analysis_threshold` fail policy checks and are emitted in SARIF:

```sh
nw policy check --include-analysis --json
nw policy sarif --include-analysis --output nightward.sarif
nw policy sarif --workspace . --include-analysis --output -
nw policy badge --workspace . --include-analysis --sarif-url https://example.invalid/nightward.sarif --output nightward-badge.json
```

SARIF analysis rules are emitted under `nightward/analyze/<rule>`.

`policy badge` emits a Shields-compatible JSON artifact plus Nightward summary fields: pass/fail, policy threshold, finding count, policy violations, ignored count, analysis signal violations, timestamp, and an optional SARIF URL. It is an artifact command, not a gate; use `policy check` to fail CI.
