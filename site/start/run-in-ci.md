# Run In CI

Use workspace mode in CI so Nightward scans the repository checkout, not the runner's HOME.

## GitHub Action

```yaml
- uses: JSONbored/nightward@v0.1.4
  with:
    mode: sarif
    output: nightward.sarif
```

Upload SARIF with GitHub's code scanning action after the Nightward step.

## Raw CLI

```sh
nw scan --workspace . --json --output nightward-scan.json
nw policy check --workspace . --include-analysis --strict --json
nw policy sarif --workspace . --include-analysis --output nightward.sarif
nw policy badge --workspace . --include-analysis --sarif-url ./nightward.sarif --output nightward-badge.json
```

## Policy Config

Run this once to inspect the default shape:

```sh
nw policy init --dry-run
```

Ignore entries must include reasons. Online-capable providers stay blocked unless policy explicitly sets `allow_online_providers: true` and selected providers are configured.
