# Provider Reference

This page is generated from `nw providers list --json`.

Nightward never installs providers. Local providers run only when selected with `--with`. Online-capable providers also require `--online` or `allow_online_providers: true` in policy config.

| Provider | Mode | Command | Default | Privacy | Capabilities |
| --- | --- | --- | --- | --- | --- |
| nightward | local/offline | built-in | yes | offline; reads only the selected HOME or workspace | MCP, dotfiles, secret-path, filesystem, and local-endpoint heuristics |
| gitleaks | local/offline | `gitleaks` | no | local command; scans selected files when explicitly run | secret pattern scanning |
| trufflehog | local/offline | `trufflehog` | no | local command; scans selected files when explicitly run | secret pattern scanning with verification disabled by default |
| semgrep | local/offline | `semgrep` | no | local command; rule packs may require network if user config chooses that | static analysis and malicious dependency rules |
| trivy | online-capable | `trivy` | no | network-capable; vulnerability database updates may contact upstream services | filesystem, dependency, IaC, and secret scanning |
| osv-scanner | online-capable | `osv-scanner` | no | network-capable; queries vulnerability data for dependency manifests | open source vulnerability matching |
| socket | online-capable | `socket` | no | network-capable; uploads dependency manifest metadata and creates a remote Socket scan artifact | remote supply-chain scan creation and malicious package signals |

## Online-Capable Providers

- `trivy`: explicit filesystem scan with JSON output. Vulnerability database activity can contact upstream services, so Nightward requires `--online`.
- `osv-scanner`: explicit source scan against vulnerability data. Nightward requires `--online`.
- `socket`: creates a remote Socket scan artifact and uploads dependency manifest metadata. Nightward does not fetch remote Socket reports in v1.
