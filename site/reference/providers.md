# Provider Reference

This page is generated from `nw providers list --json`.

Nightward never installs providers. Local providers run only when selected with `--with`. Online-capable providers also require `--online` or `allow_online_providers: true` in policy config.

| Provider | Mode | Command | Default | Install | Privacy | Capabilities |
| --- | --- | --- | --- | --- | --- | --- |
| [nightward](https://github.com/JSONbored/nightward) | local/offline | built-in | yes | built-in | offline; reads only the selected HOME or workspace | MCP, dotfiles, secret-path, filesystem, and local-endpoint heuristics |
| [gitleaks](https://github.com/gitleaks/gitleaks) | local/offline | `gitleaks` | no | [docs](https://github.com/gitleaks/gitleaks#installing) | local command; scans selected files when explicitly run | secret pattern scanning |
| [trufflehog](https://github.com/trufflesecurity/trufflehog) | local/offline | `trufflehog` | no | [docs](https://github.com/trufflesecurity/trufflehog#installation) | local command; scans selected files when explicitly run | secret pattern scanning with verification disabled by default |
| [semgrep](https://semgrep.dev/) | local/offline | `semgrep` | no | [docs](https://semgrep.dev/docs/getting-started/) | local command; rule packs may require network if user config chooses that | static analysis and malicious dependency rules |
| [trivy](https://trivy.dev/) | online-capable | `trivy` | no | [docs](https://trivy.dev/latest/getting-started/installation/) | network-capable; vulnerability database updates may contact upstream services | filesystem, dependency, IaC, and secret scanning |
| [osv-scanner](https://google.github.io/osv-scanner/) | online-capable | `osv-scanner` | no | [docs](https://google.github.io/osv-scanner/installation/) | network-capable; queries vulnerability data for dependency manifests | open source vulnerability matching |
| [socket](https://socket.dev/) | online-capable | `socket` | no | [docs](https://docs.socket.dev/docs/socket-cli) | network-capable; uploads dependency manifest metadata and creates a remote Socket scan artifact | remote supply-chain scan creation and malicious package signals |

## Online-Capable Providers

- `trivy`: explicit filesystem scan with JSON output. Vulnerability database activity can contact upstream services, so Nightward requires `--online`.
- `osv-scanner`: explicit source scan against vulnerability data. Nightward requires `--online`.
- `socket`: creates a remote Socket scan artifact and uploads dependency manifest metadata. Nightward does not fetch remote Socket reports in v1.
