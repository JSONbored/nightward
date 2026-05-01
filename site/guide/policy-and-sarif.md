# Policy And SARIF

Nightward can enforce local policy in CI while keeping runtime local and redacted.

## Policy config

```sh
nw policy init --dry-run
nw policy explain
nw policy check --config .nightward.yml --strict --json
```

Policy config supports severity thresholds, ignored finding IDs or rules with reasons, trusted commands/packages, portable path allowlists, machine-local deny paths, and SARIF naming overrides.

## SARIF

```sh
nw policy sarif --workspace . --include-analysis --output nightward.sarif
```

Use SARIF with GitHub code scanning to surface Nightward findings alongside CodeQL and other security tools.

## Analysis in policy

Analysis signals are optional in policy checks:

```sh
nw policy check --workspace . --include-analysis --strict --json
```

The default analysis engine is offline. Online-capable providers stay blocked unless explicitly enabled.
