# CLI

Nightward has two equivalent commands:

- `nightward`
- `nw`

## Core commands

```sh
nw
nw scan
nw doctor
nw analyze
nw findings list --json
nw findings explain <finding-id> --json
nw fix plan
nw fix export --format markdown
nw rules list --json
nw rules explain mcp_secret_header --json
nw adapters list --json
nw adapters explain Codex --json
nw policy check --strict --json
nw policy sarif --output nightward.sarif
```

## Workspace mode

Use workspace mode for CI, Trunk, and dotfiles repositories:

```sh
nw scan --workspace . --json
nw analyze --workspace . --json
nw policy sarif --workspace . --include-analysis --output -
```

## Scheduling

```sh
nw schedule plan --preset nightly
nw schedule install --preset nightly --dry-run
nw schedule remove --dry-run
```

Schedule install/remove are plan-only in v1. Review the generated plan before any future write-capable scheduler workflow.

## Static report

```sh
nw scan --json --output /tmp/nightward-scan.json
nw report html --input /tmp/nightward-scan.json --output /tmp/nightward-report.html
nw report diff --from /tmp/previous.json --to /tmp/nightward-scan.json
nw report html
nw report history
nw report latest
```

The HTML report is a local static file rendered from redacted scan JSON. If you omit `--input`, Nightward scans HOME. If you omit `--output`, it writes `nightward-report.html` in the current directory. Use `nw report diff --from previous.json --to current.json` for added, removed, and changed findings. Report pages include local search and filters for severity, tool, rule, and fix type.

The public demo report is generated from a committed fixture home, then scrubbed before publication:

- [Scrubbed sample scan JSON](/demo/nightward-sample-scan.json)
- [Static HTML sample report](/demo/nightward-sample-report.html)
