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
- what a backup plan would do before it writes anything

## Install

```sh
go build -o ~/.local/bin/nightward ./cmd/nightward
```

Or:

```sh
make install-local
```

## Usage

Open the TUI:

```sh
nightward
```

Scan and emit redacted JSON:

```sh
nightward scan --json
```

Check local assumptions:

```sh
nightward doctor --json
```

Generate a dry-run backup plan:

```sh
nightward plan backup --target ~/dotfiles
```

List supported adapters:

```sh
nightward adapters list
```

Generate a nightly schedule plan:

```sh
nightward schedule plan --preset nightly
```

Preview schedule install without writing:

```sh
nightward schedule install --preset nightly --dry-run
```

Preview schedule removal without writing:

```sh
nightward schedule remove --dry-run
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
go run ./cmd/nightward scan --json
go run ./cmd/nightward schedule install --preset nightly --dry-run
```

## Roadmap

- encrypted snapshots
- cross-machine diff
- Raycast extension
- GitHub Action policy checks
- Docker and Unraid dashboard
- live backup after stronger policy gates
- restore only after snapshot/rollback safety exists
