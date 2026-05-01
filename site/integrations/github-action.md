# GitHub Action

Nightward ships a composite GitHub Action for repository policy checks.

```yaml
- uses: JSONbored/nightward@v0.1.0
  with:
    mode: sarif
    workspace: .
    output: nightward.sarif
```

## Modes

- `scan`: write redacted scan JSON.
- `policy`: run policy checks and fail on violations.
- `sarif`: emit SARIF for GitHub code scanning.

## Trust boundary

The action validates relative output/config paths and keeps writes inside `GITHUB_WORKSPACE`. It treats repository content as untrusted input.
