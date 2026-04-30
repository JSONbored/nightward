---
name: nightward
description: Use Nightward to audit local AI agent/devtool config, inspect MCP risk, and generate dry-run backup or schedule plans without leaking secrets.
---

# Nightward

Use this skill when a task asks to inspect local AI agent configs, MCP servers, dotfiles backup safety, or scheduled config scans with the `nightward` CLI.

## Safe Order

1. Verify the command exists:

   ```sh
   command -v nightward || go run ./cmd/nightward --help
   ```

2. Run a read-only health check:

   ```sh
   nightward doctor --json
   ```

3. Run a redacted scan:

   ```sh
   nightward scan --json
   ```

4. Generate a dry-run backup plan:

   ```sh
   nightward plan backup --target ~/dotfiles --json
   ```

5. Preview scheduling before installing anything:

   ```sh
   nightward schedule install --preset nightly --dry-run
   ```

## Do Not

- Do not run `schedule install` or `schedule remove` without explicit user approval.
- Do not copy files from scan output manually; use the backup plan classifications.
- Do not sync `secret-auth`, `runtime-cache`, or `app-owned` items into dotfiles.
- Do not assume MCP findings are false positives; review the command, env keys, and local paths first.

## Notes

Nightward v1 is local-only and read-mostly. Scan and plan commands do not mutate user config. The only write paths are explicit schedule install/remove commands.
