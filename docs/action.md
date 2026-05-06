# GitHub Action

Nightward ships a composite GitHub Action for repository policy checks.

```yaml
jobs:
  nightward:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd
      - uses: JSONbored/nightward@v0.1.11
        with:
          mode: sarif
          output: nightward.sarif
      - uses: github/codeql-action/upload-sarif@95e58e9a2cdfd71adc6e0353d5c52f41a045d225
        with:
          sarif_file: nightward.sarif
```

Inputs:

- `mode`: `scan`, `policy`, `sarif`, or `badge`
- `config`: optional `.nightward.yml`
- `strict`: fail policy checks on medium or higher findings
- `output`: output path for scan JSON or SARIF
- `home`: optional HOME override for fixture scans
- `workspace`: optional repository/workspace path to scan; defaults to `GITHUB_WORKSPACE` unless `home` is set
- `include-analysis`: include offline analysis signals in policy or SARIF modes
- `sarif-url`: optional SARIF URL included in badge mode

Outputs:

- `findings-count`
- `policy-passed`
- `output`

The action runs Nightward locally. It does not upload findings unless your workflow separately uploads artifacts or SARIF.

`config` and `output` must be relative paths inside `GITHUB_WORKSPACE`. The action rejects absolute paths, parent traversal, backslashes, and newline characters before invoking Nightward.

For repository CI, the action defaults to `GITHUB_WORKSPACE`; pass `workspace` explicitly when you want to scan a narrower checkout path:

```yaml
- uses: JSONbored/nightward@v0.1.11
  with:
    mode: sarif
    workspace: ${{ github.workspace }}
    include-analysis: "true"
    output: nightward.sarif
```

To publish a small badge JSON artifact alongside SARIF:

```yaml
- uses: JSONbored/nightward@v0.1.11
  with:
    mode: badge
    workspace: ${{ github.workspace }}
    include-analysis: "true"
    output: nightward-badge.json
    sarif-url: https://github.com/JSONbored/nightward/security/code-scanning?query=tool%3ANightward
- uses: actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a
  with:
    name: nightward-badge
    path: nightward-badge.json
```
