# Troubleshooting

## `nw` Is Not Found

Use the canonical command or install locally:

```sh
nightward --help
make install-local
```

The npm package exposes both `nightward` and `nw`.

## Provider Is Missing

Run:

```sh
nw providers doctor --with gitleaks,trufflehog,semgrep
```

Nightward only uses provider binaries already available on `PATH`. Online-capable providers stay blocked without `--online`.

## Report Output Is Empty

Check whether you scanned HOME or a workspace:

```sh
nw scan --json
nw scan --workspace . --json
```

Workspace mode intentionally ignores HOME-only config.

## Policy Fails In CI

Use `nw policy explain`, inspect `.nightward.yml`, and add reasoned ignores only for reviewed findings. Do not lower thresholds just to make CI green.
