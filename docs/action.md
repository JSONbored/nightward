# GitHub Action

Nightward ships a composite GitHub Action for repository policy checks.

```yaml
jobs:
  nightward:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd
      - uses: JSONbored/nightward@v0.1.0
        with:
          mode: sarif
          output: nightward.sarif
      - uses: github/codeql-action/upload-sarif@95e58e9a2cdfd71adc6e0353d5c52f41a045d225
        with:
          sarif_file: nightward.sarif
```

Inputs:

- `mode`: `scan`, `policy`, or `sarif`
- `config`: optional `.nightward.yml`
- `strict`: fail policy checks on medium or higher findings
- `output`: output path for scan JSON or SARIF
- `home`: optional HOME override for fixture scans

Outputs:

- `findings-count`
- `policy-passed`
- `output`

The action runs Nightward locally. It does not upload findings unless your workflow separately uploads artifacts or SARIF.
