# Report History

Nightward can compare scan reports without mutating live config.

## Write Timestamped Reports

```sh
nw scan --json --output ~/.local/state/nightward/reports/current.json
nw report history
nw report latest
```

## Compare Reports

```sh
nw report diff --from previous.json --to current.json
nw report diff --from previous.json --to current.json --json
```

`nw report diff` compares two explicit report files. The diff groups added, removed, and changed findings by stable finding IDs. Legacy reports without IDs are compared with generated keys from finding content.

## Render Review Artifacts

```sh
nw report html
nw report html --input current.json --output current.html
nw report index
```

`nw report html` scans HOME by default or renders the explicit `--input` scan JSON. HTML reports include local finding search, severity/tool/rule/fix filters, collapsible evidence, and remediation groups. The filter controls run entirely inside the static HTML file.

The history index summarizes local report files with finding totals, highest severity, severity badges, and deltas against the next-newer report.
