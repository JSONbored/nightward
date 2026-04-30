# Nightward

Nightward is a local-first TUI and CLI for watching AI agent/devtool state before it leaks into dotfiles.

It scans common Codex, Claude, Cursor, Windsurf, VS Code, Raycast, and MCP config locations; classifies what is portable versus local-only or secret; and builds dry-run backup plans for private dotfiles repos.

V1 is read-only except explicit schedule install/remove commands.

## Why

AI coding tools scatter useful state across config files, MCP server definitions, skills, rules, commands, extension settings, credentials, caches, and app-owned databases. Syncing all of that blindly is fragile and unsafe.

Nightward gives you a local inventory first:

- what exists
- what is safe to sync
- what needs review
- what should never be committed
- which MCP configs deserve security attention
- what exact remediation plan is safe to consider
- what a backup plan would do before it writes anything

## Install

```sh
make install-local
```

This installs both commands:

- `nightward`: canonical project command
- `nw`: short alias for frequent terminal/TUI use

## Usage

Open the TUI:

```sh
nightward
# or
nw
```

Scan and emit redacted JSON:

```sh
nightward scan --json
# or
nw scan --json
```

Check local assumptions:

```sh
nightward doctor --json
# or
nw doctor --json
```

Generate a dry-run backup plan:

```sh
nightward plan backup --target ~/dotfiles
# or
nw plan backup --target ~/dotfiles
```

List supported adapters:

```sh
nightward adapters list
# or
nw adapters list
```

List findings:

```sh
nightward findings list
# or
nw findings list --json
```

Explain a finding:

```sh
nw findings explain mcp_unpinned_package-abc123
```

Generate a read-only fix plan:

```sh
nw fix plan --all --json
nw fix plan --rule mcp_secret_env
nw fix export --format markdown
```

Run policy checks or generate SARIF:

```sh
nw policy check --strict --json
nw policy sarif --output nightward.sarif
```

Generate a nightly schedule plan:

```sh
nightward schedule plan --preset nightly
# or
nw schedule plan --preset nightly
```

Preview schedule install without writing:

```sh
nightward schedule install --preset nightly --dry-run
# or
nw schedule install --preset nightly --dry-run
```

Preview schedule removal without writing:

```sh
nightward schedule remove --dry-run
# or
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

## MCP/Security Checks

Nightward inspects JSON and TOML MCP config shapes where possible and flags:

- unpinned package execution through tools such as `npx`, `uvx`, or `pipx`
- shell-mediated MCP commands
- sensitive environment key references
- broad filesystem access
- local credential path references
- MCP servers that need manual review

Reports redact values and expose only path, tool, classification, risk, reason, and recommended action.

## Fix Plans

Nightward does not mutate agent configs in this release. "Autofix" means structured, reviewable fix plans:

- `pin-package`: pin `npx`, `uvx`, or `pipx` package execution when the package name is parseable
- `externalize-secret`: move inline secret values out of agent config and keep only env key names or setup docs
- `replace-shell-wrapper`: replace simple shell passthroughs with direct executable invocation
- `narrow-filesystem`: replace broad filesystem access with explicit paths after human review
- `manual-review`: inspect unsupported, ambiguous, or high-risk config manually
- `ignore-with-reason`: keep an advisory finding only after documenting why it is expected

Every fix plan includes confidence, risk, review requirements, redacted evidence, impact, and steps. Secret values are never emitted in scan JSON, fix-plan JSON, Markdown exports, SARIF, or TUI detail text.

## TUI

The default `nightward` / `nw` command opens a Bubble Tea TUI with:

- dashboard metrics and schedule status
- inventory by tool/classification
- findings list with severity/tool/rule filters
- selected-finding detail with evidence, impact, fix plan, and why it matters
- fix-plan summary grouped as safe, review, and blocked
- backup plan preview

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
go test ./...
go run ./cmd/nightward --help
go run ./cmd/nw --help
go run ./cmd/nightward scan --json
go run ./cmd/nw findings list --json
go run ./cmd/nw fix plan --all --json
go run ./cmd/nw policy sarif --output /tmp/nightward.sarif
go run ./cmd/nightward schedule install --preset nightly --dry-run
```

## Project Docs

- [Security policy](SECURITY.md)
- [Contributing guide](CONTRIBUTING.md)
- [Roadmap](ROADMAP.md)
- [CI/security notes](docs/ci-security.md)
- [Screenshot/GIF capture plan](docs/screenshots.md)
