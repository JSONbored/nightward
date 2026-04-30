# Analysis

Nightward analysis turns scan findings and classified paths into explainable risk signals.

Default behavior is offline:

```sh
nw analyze --all --json
nw analyze --all --workspace . --json
```

Analysis output includes:

- `subjects`: findings, inventory items, or packages under review
- `signals`: severity, category, confidence, evidence, and recommended action
- `providers`: built-in and optional provider status
- `summary`: counts by severity, category, and provider

Nightward avoids "safe" claims. A clean analysis means Nightward found no known risk signals from the enabled providers, not that the subject is trustworthy.

## Providers

The built-in `nightward` provider is always offline and enabled by default. Optional providers are discovered but not installed or run automatically:

- `gitleaks`
- `trufflehog`
- `semgrep`
- `trivy`
- `osv-scanner`
- `socket`

Check provider posture:

```sh
nw providers list --json
nw providers doctor --json
nw providers doctor --with socket --json
```

Online-capable providers remain blocked unless explicitly allowed:

```sh
nw providers doctor --with socket --online --json
```

Policy/SARIF config can opt into provider posture:

```yaml
include_analysis: true
analysis_threshold: high
analysis_providers:
  - socket
allow_online_providers: false
```

## Policy And SARIF

Analysis signals are advisory unless included explicitly:

```sh
nw policy check --include-analysis --json
nw policy sarif --include-analysis --output nightward.sarif
nw policy sarif --workspace . --include-analysis --output -
```

SARIF analysis rules are emitted under `nightward/analyze/<rule>`.
