# Nightward

[![CI](https://github.com/JSONbored/nightward/actions/workflows/ci.yml/badge.svg)](https://github.com/JSONbored/nightward/actions/workflows/ci.yml)
[![Nightward Policy](https://github.com/JSONbored/nightward/actions/workflows/nightward-policy.yml/badge.svg)](https://github.com/JSONbored/nightward/actions/workflows/nightward-policy.yml)
[![OpenSSF Scorecard](https://github.com/JSONbored/nightward/actions/workflows/scorecard.yml/badge.svg)](https://github.com/JSONbored/nightward/actions/workflows/scorecard.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

Nightward is a local-first TUI and CLI for auditing AI agent and devtool state before it leaks into dotfiles.

It scans common Codex, Claude, Cursor, Windsurf, VS Code, Raycast, JetBrains, Zed, Continue, Cline/Roo, Aider, OpenCode, Goose, LM Studio, Ollama/Open WebUI, Neovim, and MCP config locations; classifies what is portable versus local-only or secret; highlights MCP security findings; and produces redacted fix plans, fix previews, SARIF policy output, snapshot plans, and dry-run backup plans.

Nightward does not mutate agent configs. It only writes explicit report/SARIF files when requested and user-level schedule files through explicit schedule install/remove commands.

## Why

AI coding tools scatter useful state across config files, MCP server definitions, skills, rules, commands, extension settings, credentials, caches, and app-owned databases. Blindly syncing all of it is fragile and unsafe.

Nightward answers the practical questions first:

- What exists on this machine?
- What is portable enough for a private dotfiles repo?
- What is machine-local, app-owned, runtime cache, or credential material?
- Which MCP configs deserve security review?
- What exact remediation plan should I consider before syncing?
- What would a backup plan include, review, or exclude before it writes anything?

## Highlights

- Bubble Tea TUI with dashboard, inventory, findings, fix plan, and backup preview tabs.
- `nightward` canonical command plus `nw` short alias.
- Redacted JSON for automation and CI.
- MCP findings for unpinned package execution, shell wrappers, sensitive env keys, broad filesystem access, token paths, parse failures, and unknown server shapes.
- Plan-only remediation metadata: fix kind, confidence, risk, review requirement, impact, and steps.
- SARIF output for GitHub code scanning.
- Optional `.nightward.yml` policy config with reason-required ignores.
- Redacted patch previews for parseable MCP config fixes.
- Read-only snapshot plan/diff commands.
- Reusable GitHub Action for scan, policy, and SARIF modes.
- Read-only Raycast extension for Dashboard, Findings, Explain Finding, Fix Plan export, and report-folder access.
- User-level nightly scan scheduling for macOS launchd, Linux systemd user timers, and cron text fallback.
- No telemetry, no cloud dashboard, no network calls from Nightward runtime, and no live config mutation.

## Install

```sh
make install-local
```

This installs:

- `nightward`: canonical project command
- `nw`: short alias for frequent terminal/TUI use

## Quick Start

Open the TUI:

```sh
nw
```

Scan and emit redacted JSON:

```sh
nw scan --json
```

Check local assumptions:

```sh
nw doctor --json
```

List and explain findings:

```sh
nw findings list
nw findings explain mcp_unpinned_package-abc123
```

Generate a plan-only fix report:

```sh
nw fix plan --all --json
nw fix plan --rule mcp_secret_env
nw fix preview --rule mcp_secret_env --format diff
nw fix preview --all --format markdown
nw fix export --format markdown
```

Create or explain policy config:

```sh
nw policy init --dry-run
nw policy explain
nw policy check --config .nightward.yml --strict --json
```

Generate a dry-run backup plan:

```sh
nw plan backup --target ~/dotfiles
```

Generate read-only snapshot plans and compare them:

```sh
nw snapshot plan --target ~/nightward-snapshots --json
nw snapshot diff --from before.json --to after.json --json
```

Run policy checks or generate SARIF:

```sh
nw policy check --strict --json
nw policy sarif --output nightward.sarif
```

Preview scheduled nightly scans:

```sh
nw schedule plan --preset nightly
nw schedule install --preset nightly --dry-run
nw schedule remove --dry-run
```

## Classification Model

Nightward classifies discovered state as:

- `portable`: usually safe to sync after review
- `machine-local`: tied to local paths, identities, or machine assumptions
- `secret-auth`: credentials or auth material; excluded by default
- `runtime-cache`: generated runtime data; excluded by default
- `app-owned`: databases, extension binaries, encrypted app state, or caches owned by another app
- `unknown`: found state Nightward cannot classify confidently

Backup plans include portable items, mark machine-local/unknown items for review, and exclude secret/auth, runtime-cache, and app-owned state by default.

## Fix Plan Model

Nightward does not apply fixes yet. "Autofix" currently means structured, reviewable fix plans:

- `pin-package`: pin `npx`, `uvx`, or `pipx` package execution when the package name is parseable
- `externalize-secret`: move inline secret values out of agent config and keep only env key names or setup docs
- `replace-shell-wrapper`: replace simple shell passthroughs with direct executable invocation
- `narrow-filesystem`: replace broad filesystem access with explicit paths after human review
- `manual-review`: inspect unsupported, ambiguous, or high-risk config manually
- `ignore-with-reason`: keep an advisory finding only after documenting why it is expected

Secret values are never emitted in scan JSON, findings output, fix-plan JSON, Markdown exports, SARIF, or TUI detail text.

## TUI

The default `nightward` / `nw` command opens the TUI:

- Dashboard: scan counts and schedule status
- Inventory: discovered paths by tool, classification, and risk
- Findings: severity/tool/rule filters with a selected-finding detail pane
- Fix Plan: safe/review/blocked remediation groups
- Backup Plan: private-dotfiles dry-run preview

Keyboard shortcuts:

- `1`-`5`: switch tabs
- arrow keys or `h`/`j`/`k`/`l`: navigate
- `/`: search findings
- `s`, `t`, `r`: cycle finding severity, tool, and rule filters
- `x`: clear finding filters
- `c`, `e`, `o`: show copy/export/docs action hints
- `?`: help
- `q` or `esc`: quit

## GitHub Action

Nightward can run as a local GitHub Action in scan, policy, or SARIF mode:

```yaml
- uses: JSONbored/nightward@v0.1.0
  with:
    mode: sarif
    output: nightward.sarif
```

See [docs/action.md](docs/action.md) for inputs, outputs, and SARIF upload examples.

## Raycast Extension

The read-only Raycast extension lives in [integrations/raycast](integrations/raycast).

```sh
cd integrations/raycast
npm ci
npm run build
npm run dev
```

Commands:

- `Nightward Dashboard`
- `Nightward Findings`
- `Explain Nightward Finding`
- `Export Nightward Fix Plan`
- `Open Nightward Reports`

The extension shells out to `nw` or `nightward`, renders redacted output, copies an explicitly requested fix-plan export, and opens the local reports folder. It does not mutate agent configs or install schedules.

See [docs/raycast-extension.md](docs/raycast-extension.md) for preferences, validation, and read-only boundaries.

## Scheduling

The `nightly` preset runs:

```sh
nightward scan --json --output-dir ~/.local/state/nightward/reports
```

Supported user-level schedule targets:

- macOS: `launchd` user agent
- Linux: systemd user timer
- Other platforms: generated cron text only

Scheduled scans never copy secrets, mutate dotfiles, restore files, or push to Git.

## Development

```sh
make test
make test-race
make test-junit
make trunk-flaky-validate
make trunk-check
make raycast-verify
make verify
go run ./cmd/nightward --help
go run ./cmd/nw --help
go run ./cmd/nw scan --json
go run ./cmd/nw findings list --json
go run ./cmd/nw fix plan --all --json
go run ./cmd/nw fix preview --all --format markdown
go run ./cmd/nw policy sarif --output /tmp/nightward.sarif
go run ./cmd/nw schedule install --preset nightly --dry-run
```

Local security checks used by maintainers:

```sh
trunk check --show-existing --all
gitleaks detect --source . --no-git --redact
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

## Project Docs

- [Security policy](SECURITY.md)
- [Contributing guide](CONTRIBUTING.md)
- [Code of conduct](CODE_OF_CONDUCT.md)
- [Support](SUPPORT.md)
- [Roadmap](ROADMAP.md)
- [Adapters](docs/adapters.md)
- [Remediation](docs/remediation.md)
- [Testing](docs/testing.md)
- [Dependency maintenance](docs/dependency-maintenance.md)
- [GitHub Action](docs/action.md)
- [Raycast extension](docs/raycast-extension.md)
- [CI/security notes](docs/ci-security.md)
- [Privacy model](docs/privacy-model.md)
- [Screenshot/GIF capture plan](docs/screenshots.md)

## Contributors

[![Contributors](https://contrib.rocks/image?repo=JSONbored/nightward)](https://github.com/JSONbored/nightward/graphs/contributors)

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=JSONbored/nightward&type=Date)](https://www.star-history.com/#JSONbored/nightward&Date)

## License

MIT
