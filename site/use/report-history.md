# Report History

Nightward can compare scan reports without mutating live config.

## Write Timestamped Reports

```sh
nw scan --json --output-dir ~/.local/state/nightward/reports
nw report history
```

## Compare Reports

```sh
nw report diff --from previous.json --to current.json
nw report diff --from previous.json --to current.json --json
```

The diff groups added, removed, changed, and unchanged findings by stable finding IDs. Legacy reports without IDs are compared with generated keys from finding content.

## Render Review Artifacts

```sh
nw report html --input current.json --previous previous.json --output current.html
nw report index --dir ~/.local/state/nightward/reports --output index.html
```

HTML reports include severity sections, collapsible evidence, remediation groups, and optional report-to-report changes.
