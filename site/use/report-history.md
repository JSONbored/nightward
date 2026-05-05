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
nw tui --from previous.json --to current.json
```

`nw report diff` compares two explicit report files. The diff groups added, removed, and changed findings by stable finding IDs. Legacy reports without IDs are compared with generated keys from finding content. `nw tui --from ... --to ...` opens the same comparison as a read-only terminal review flow.

## Render Review Artifacts

```sh
nw report html
nw report html --input current.json --output current.html
nw report html --from previous.json --to current.json --output compare.html
nw report index
```

`nw report html` scans HOME by default, renders the explicit `--input` scan JSON, or renders the `--from`/`--to` comparison alongside the latest report content.

The history index summarizes local report files with finding totals, highest severity, severity badges, and deltas against the next-newer report.
