# Provider Reference

This page is generated from `nw providers list --json`.

Nightward never installs providers. Local providers run only when selected with `--with`. Online-capable providers also require `--online` or `allow_online_providers: true` in policy config.

| Provider | Mode | Command | Default | Install | Privacy | Capabilities |
| --- | --- | --- | --- | --- | --- | --- |
| [nightward](https://github.com/JSONbored/nightward) | local/offline | built-in | yes | built-in | local-only | inventory, MCP config posture, dotfiles safety |
| [gitleaks](https://github.com/gitleaks/gitleaks) | local/offline | `gitleaks` | no | [docs](https://github.com/gitleaks/gitleaks#installing) | local command; no network enabled by Nightward | secret scanning |
| [trufflehog](https://github.com/trufflesecurity/trufflehog) | local/offline | `trufflehog` | no | [docs](https://github.com/trufflesecurity/trufflehog#installation) | local command; no network enabled by Nightward | secret scanning |
| [semgrep](https://semgrep.dev/) | local/offline | `semgrep` | no | [docs](https://semgrep.dev/docs/getting-started/) | local command; no network enabled by Nightward | local rule scanning |
| [trivy](https://trivy.dev/) | online-capable | `trivy` | no | [docs](https://trivy.dev/latest/getting-started/installation/) | online-capable; blocked unless explicitly enabled | filesystem vulnerability, secret, and misconfig scanning |
| [osv-scanner](https://google.github.io/osv-scanner/) | online-capable | `osv-scanner` | no | [docs](https://google.github.io/osv-scanner/installation/) | online-capable; blocked unless explicitly enabled | dependency vulnerability scanning |
| [socket](https://socket.dev/) | online-capable | `socket` | no | [docs](https://docs.socket.dev/docs/socket-cli) | online-capable; creates a remote Socket scan artifact | dependency risk metadata and Socket scan creation |

## Online-Capable Providers

- `trivy`: explicit filesystem scan with JSON output. Vulnerability database activity can contact upstream services, so Nightward requires `--online`.
- `osv-scanner`: explicit source scan against vulnerability data. Nightward requires `--online`.
- `socket`: creates a remote Socket scan artifact and uploads dependency manifest metadata. Nightward does not fetch remote Socket reports in v1.
