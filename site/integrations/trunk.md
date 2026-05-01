# Trunk

Nightward includes an importable Trunk plugin definition after release tags exist.

```sh
trunk plugins add --id nightward https://github.com/JSONbored/nightward v0.1.0
trunk check enable nightward-policy
```

## Linters

- `nightward-policy`: emits SARIF from workspace policy checks.
- `nightward-analyze`: emits SARIF with offline analysis signals included.

Both linters are security-focused and read-only.
