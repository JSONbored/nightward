# JSON Output Reference

Nightward emits redacted JSON for automation.

Public JSON objects include `schema_version` unless they predate the v0.1.4 schema contract. Pre-1.0 schema changes should stay additive whenever possible.

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

## Report Diff

```sh
nw report diff --from previous.json --to current.json --json
nw report history --json
nw report latest --json
```

Report diff output includes:

- `added_findings`
- `removed_findings`
- `changed_findings`
- summary counts for added, removed, changed, and unchanged findings

Report history records include `path`, `report_name`, `mod_time`, `findings`, `highest_severity`, `findings_by_severity`, and `size_bytes`.

## Badge

```sh
nw policy badge --output -
```

The badge artifact keeps the Shields-compatible `schemaVersion` key and adds Nightward policy fields such as `passed`, `threshold`, `total_findings`, violation counts, and an optional SARIF URL.
