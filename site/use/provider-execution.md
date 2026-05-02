# Provider Execution

Nightward has built-in offline heuristics and optional explicit providers.

## Local Providers

These run only when selected with `--with`:

- `gitleaks`
- `trufflehog`
- `semgrep`

```sh
nw providers doctor --with gitleaks,trufflehog,semgrep
nw analyze --workspace . --with gitleaks,trufflehog,semgrep --json
```

Nightward does not install tools. It discovers them on `PATH`, runs bounded commands, parses supported JSON shapes, and redacts provider-derived evidence before emitting JSON, SARIF, TUI, Raycast, or badge output.

## Online-Capable Providers

These require both provider selection and an online gate:

```sh
nw analyze --workspace . --with trivy,osv-scanner,socket --online --json
```

| Provider | Behavior |
| --- | --- |
| `trivy` | Runs a filesystem scan with JSON output. Vulnerability database behavior can contact upstream services. |
| `osv-scanner` | Runs source scanning against vulnerability data. |
| `socket` | Creates a remote Socket scan artifact and uploads dependency manifest metadata. Nightward does not fetch remote Socket reports in v1. |

Use `allow_online_providers: true` only in policy files where that network behavior is intended.
