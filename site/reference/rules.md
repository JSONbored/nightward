# Rule Reference

This page is generated from `nw rules list --json`.

| Rule | Severity | Docs | Fix | Summary |
| --- | --- | --- | --- | --- |
| `mcp_secret_env` | critical | [docs](https://jsonbored.github.io/nightward/guide/remediation) | `externalize-secret` | MCP server stores a sensitive environment variable inline |
| `mcp_secret_header` | critical | [docs](https://jsonbored.github.io/nightward/guide/remediation) | `externalize-secret` | MCP server stores a sensitive header inline |
| `mcp_unpinned_package` | high | [docs](https://jsonbored.github.io/nightward/guide/mcp-security) | `pin-package` | MCP server runs a package executor without an obvious pin |
| `mcp_shell_wrapper` | high | [docs](https://jsonbored.github.io/nightward/guide/mcp-security) | `replace-shell-wrapper` | MCP server runs through a shell wrapper |
| `mcp_local_endpoint` | medium | [docs](https://jsonbored.github.io/nightward/guide/mcp-security) | `manual-review` | MCP server references a machine-local endpoint |
| `mcp_broad_filesystem` | high | [docs](https://jsonbored.github.io/nightward/guide/mcp-security) | `narrow-filesystem` | MCP server can access a broad filesystem path |
| `mcp_local_token_path` | high | [docs](https://jsonbored.github.io/nightward/guide/privacy-model) | `manual-review` | MCP server references a local credential path |
| `mcp_unknown_command` | info | [docs](https://jsonbored.github.io/nightward/reference/rules) | `manual-review` | MCP server should be reviewed |
| `config_parse_failed` | medium | [docs](https://jsonbored.github.io/nightward/use/troubleshooting) | `manual-review` | Nightward could not parse a config file |
| `config_symlink` | info | [docs](https://jsonbored.github.io/nightward/guide/privacy-model) | `manual-review` | Config file is a symbolic link |
