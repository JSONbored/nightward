# Adapters

Adapters declare expected local paths for AI agents, editors, MCP clients, and adjacent devtools. They classify state; they do not copy, restore, upload, or mutate files.

## Current Adapter Families

- Codex
- Claude and Claude Code
- Cursor
- Windsurf
- VS Code
- Raycast
- JetBrains IDEs
- Zed
- Continue
- Cline and Roo Code
- Aider
- OpenCode
- Goose
- LM Studio
- Ollama and Open WebUI
- Neovim
- Generic MCP config files

## Classification Rules

- `portable`: candidate for private dotfiles after review.
- `machine-local`: tied to machine paths, identities, services, or provider setup.
- `secret-auth`: credentials or auth material; exclude by default.
- `runtime-cache`: generated state; exclude by default.
- `app-owned`: app databases, model stores, extension binaries, encrypted stores, or sessions.
- `unknown`: discovered state that needs a human decision.

## Adding An Adapter

New adapters must include:

- temporary HOME fixture tests
- backup-plan mapping tests
- no-write coverage through CLI or scanner tests
- redaction tests for config contents when files are parsed
- conservative classification for app databases, auth, model blobs, caches, and extension storage

If an adapter parses executable config, its findings should include remediation metadata and never emit secret values.

MCP adapters should treat URL-shaped remote servers as first-class config, not as unknown commands. They should still flag sensitive headers and local/private endpoints without printing header values or URL path/query details.
