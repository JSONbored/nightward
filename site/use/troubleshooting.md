# Troubleshooting

## `nw` Is Not Found

Install the release-backed npm launcher, then open the TUI or run a CLI command:

```sh
npm install -g @jsonbored/nightward
nw
nw scan --json
```

The npm package exposes both `nightward` and `nw`. If you are working from a source checkout instead, run `make install-local` to install both binaries into `~/.local/bin` by default.

## Provider Is Missing

Run:

```sh
nw providers doctor --with gitleaks,trufflehog,semgrep,syft
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
