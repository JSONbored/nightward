# Rules Reference

Nightward rules are intentionally conservative. A finding asks for review; it does not claim that a tool is malicious.

## MCP rules

- `mcp_server_review`
- `mcp_unpinned_package`
- `mcp_secret_env`
- `mcp_secret_header`
- `mcp_shell_command`
- `mcp_broad_filesystem`
- `mcp_local_endpoint`
- `mcp_local_token_path`
- `mcp_symlink_config`
- `mcp_unknown_command`
- `mcp_parse_failed`

## Analysis categories

- `supply-chain`
- `secrets-exposure`
- `filesystem-scope`
- `network-exposure`
- `execution-risk`
- `machine-locality`
- `app-state`
- `unknown`

Use these commands for rule metadata and local finding detail:

```sh
nw rules list
nw rules explain mcp_secret_header
nw findings explain <id>
nw trust explain <id>
```
