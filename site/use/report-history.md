# Report History

Nightward can compare scan reports without mutating live config.

## Write Timestamped Reports

```sh
nw scan --json --output-dir ~/.local/state/nightward/reports
nw report history
nw report latest
```

## Compare Reports

```sh
nw report changes
nw report diff
nw report diff --from previous.json --to current.json
nw report diff --from previous.json --to current.json --json
```

`nw report changes` and `nw report diff` compare the latest two saved reports by default. Pass explicit paths when you want to compare two specific files. The diff groups added, removed, changed, and unchanged findings by stable finding IDs. Legacy reports without IDs are compared with generated keys from finding content.

## Render Review Artifacts

```sh
nw report html
nw report html --input current.json --previous previous.json --output current.html
nw report index --dir ~/.local/state/nightward/reports --output index.html
```

`nw report html` uses the latest saved report by default, auto-compares against the previous saved report when available, and writes a private HTML file under the Nightward report directory. HTML reports include local finding search, severity/tool/rule/fix filters, collapsible evidence, remediation groups, and optional report-to-report changes. The filter controls run entirely inside the static HTML file.

The history index summarizes local report files with finding totals, highest severity, severity badges, and deltas against the next-newer report.
