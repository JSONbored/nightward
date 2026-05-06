# Provider Execution

Nightward has built-in offline heuristics and optional explicit providers.

## Local Providers

These run only when selected with `--with`:

- [Gitleaks](https://github.com/gitleaks/gitleaks): secret scanning.
- [TruffleHog](https://github.com/trufflesecurity/trufflehog): secret scanning with verification disabled by Nightward's default runner.
- [Semgrep](https://semgrep.dev/): static analysis using explicit local config.
- [Syft](https://oss.anchore.com/syft/): local SBOM and package inventory.

```sh
nw providers doctor --with gitleaks,trufflehog,semgrep,syft
nw analyze --workspace . --with gitleaks,trufflehog,semgrep,syft --json
```

Nightward discovers providers on `PATH`, marks unselected optional providers as `skipped`, runs bounded commands only when selected, parses supported JSON shapes, and redacts provider-derived evidence before emitting JSON, SARIF, TUI, Raycast, MCP, policy, badge, or HTML output. Timeout and output-cap failures are provider warnings, not clean results.

Known provider installs are available through the shared action registry and require disclosure acceptance plus `--confirm`, TUI confirmation, or Raycast confirmation:

```sh
nw providers install gitleaks --confirm
nw providers enable gitleaks --confirm
```

## Online-Capable Providers

These require both provider selection and an online gate:

```sh
nw analyze --workspace . --with trivy,osv-scanner,grype,scorecard,socket --online --json
```

| Provider | Behavior |
| --- | --- |
| [`trivy`](https://trivy.dev/) | Runs a filesystem scan with JSON output. Vulnerability database behavior can contact upstream services. |
| [`osv-scanner`](https://google.github.io/osv-scanner/) | Runs source scanning against vulnerability data. |
| [`grype`](https://oss.anchore.com/grype/) | Runs filesystem/SBOM vulnerability scanning. Vulnerability database behavior can contact upstream services. |
| [`scorecard`](https://github.com/ossf/scorecard) | Runs repository-trust checks against the git remote or `NIGHTWARD_SCORECARD_REPO`. |
| [`socket`](https://socket.dev/) | Creates a remote Socket scan artifact and uploads dependency manifest metadata. Nightward does not fetch remote Socket reports in v1. |

Use `allow_online_providers: true` only in policy files where that network behavior is intended.

## Raycast Provider Doctor

The Raycast Provider Doctor mirrors this model:

- enable or disable selected providers for Raycast Analysis;
- keep online-capable providers blocked until the extension preference allows them;
- show install commands and upstream docs when a provider is missing.

Raycast provider actions use the same action registry and confirmation prompts as the CLI/TUI.
