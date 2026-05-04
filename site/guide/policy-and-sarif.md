# Policy And SARIF

Nightward can enforce local policy in CI while keeping runtime local and redacted.

## Policy config

```sh
nw policy init
nw policy explain
nw policy check --config .nightward.yml --strict --json
```

Policy config supports severity thresholds, reason-required finding and rule ignores, optional analysis thresholds, provider selection, and online-provider gating.

## SARIF

```sh
nw policy sarif --workspace . --include-analysis --output nightward.sarif
```

Use SARIF with GitHub code scanning to surface Nightward findings alongside CodeQL and other security tools.

## Badge artifact

Generate a small JSON artifact for dashboards, README automation, or release evidence:

```sh
nw policy badge --workspace . --include-analysis --sarif-url https://github.com/JSONbored/nightward/security/code-scanning --output nightward-badge.json
```

The badge artifact includes pass/fail status, policy threshold, finding counts, signal counts, and the optional SARIF URL. It is a JSON status artifact, not a hosted service.

## Analysis in policy

Analysis signals are optional in policy checks:

```sh
nw policy check --workspace . --include-analysis --strict --json
```

The default analysis engine is offline. Online-capable providers stay blocked unless explicitly enabled.
