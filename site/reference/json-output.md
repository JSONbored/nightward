# JSON Output Reference

Nightward emits redacted JSON for automation.

## Scan

```sh
nw scan --json
```

The scan summary separates inventory counts from finding counts:

- `items_by_classification`
- `items_by_risk`
- `items_by_tool`
- `findings_by_severity`
- `findings_by_rule`
- `findings_by_tool`

## Findings

```sh
nw findings list --json
nw findings explain <finding-id> --json
```

Findings include rule, severity, evidence, impact, recommendation, fix metadata, and redacted patch hints where safe.

## Policy

```sh
nw policy check --strict --json
```

Policy output includes pass/fail status, threshold, violations, ignored findings, and optional analysis signal violations.
